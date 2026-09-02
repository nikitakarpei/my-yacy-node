//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	streamAppearanceWait = 60 * time.Second
	offerWait            = 90 * time.Second
	silenceWait          = 20 * time.Second
)

func startScrapeStack(t *testing.T, ctx context.Context) (jetstream.JetStream, jetstream.Consumer) {
	t.Helper()
	network := dockernetwork.New(t, ctx)
	natsURL := natsjetstream.Start(t, ctx, network.Name)
	startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	startPageScrape(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	awaitStream(t, ctx, js, pagescrapecontract.ScrapeRequestsStreamName)
	offers := awaitStream(t, ctx, js, pagescrapecontract.ScrapePageOffersStreamName)

	consumer, err := offers.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:   "corpustest",
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create page offer consumer: %v", err)
	}
	return js, consumer
}

func connectJetStream(t *testing.T, url string) jetstream.JetStream {
	t.Helper()
	conn, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(conn.Close)
	js, err := jetstream.New(conn)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	return js
}

func awaitStream(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	name string,
) jetstream.Stream {
	t.Helper()
	var stream jetstream.Stream
	appeared := pollwait.For(streamAppearanceWait, func() bool {
		opened, err := js.Stream(ctx, name)
		if err != nil {
			return false
		}
		stream = opened
		return true
	})
	if !appeared {
		t.Fatalf("stream %s did not appear within %s", name, streamAppearanceWait)
	}
	return stream
}

func publishScrapeRequest(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	pageURL string,
) {
	t.Helper()
	pageCanonicalURL := canonicalurltest.CanonicalURLOf(t, pageURL)
	data, err := pagescrapecontract.MarshalScrapeRequest(pagescrapecontract.ScrapeRequest{
		PageURL:  pageCanonicalURL,
		FetchURL: pageCanonicalURL,
	})
	if err != nil {
		t.Fatalf("marshal scrape request: %v", err)
	}
	if _, err := js.Publish(ctx, pagescrapecontract.ScrapeRequestSubject, data); err != nil {
		t.Fatalf("publish scrape request: %v", err)
	}
}

func nextMessage(t *testing.T, consumer jetstream.Consumer, within time.Duration) jetstream.Msg {
	t.Helper()
	message, err := consumer.Next(jetstream.FetchMaxWait(within))
	if err != nil {
		return nil
	}
	if err := message.Ack(); err != nil {
		t.Fatalf("acknowledge %s: %v", message.Subject(), err)
	}
	return message
}
