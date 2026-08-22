//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/scraperequestcontract"
)

const (
	ordersSubject        = "yacy.crawl.orders"
	streamAppearanceWait = 60 * time.Second
	messageArrivalWait   = 60 * time.Second
)

func connectJetStream(t *testing.T, url string) jetstream.JetStream {
	t.Helper()
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("connect nats: %v", err)
	}
	t.Cleanup(nc.Close)
	js, err := jetstream.New(nc)
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
		found, err := js.Stream(ctx, name)
		if err != nil {
			return false
		}
		stream = found
		return true
	})
	if !appeared {
		t.Fatalf("crawler did not create the %s stream within %s", name, streamAppearanceWait)
	}
	return stream
}

func fetchOneScrapeRequest(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
) scraperequestcontract.ScrapeRequest {
	t.Helper()
	stream := awaitStream(t, ctx, js, scraperequestcontract.ScrapeRequestsStreamName)
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create scrape request consumer: %v", err)
	}
	msg, err := consumer.Next(jetstream.FetchMaxWait(messageArrivalWait))
	if err != nil {
		t.Fatalf("fetch scrape request: %v", err)
	}
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(msg.Data())
	if err != nil {
		t.Fatalf("decode scrape request: %v", err)
	}
	if err := msg.Ack(); err != nil {
		t.Fatalf("ack scrape request: %v", err)
	}
	return scrapeRequest
}

func fetchScrapeRequestForDurable(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	durable string,
) scraperequestcontract.ScrapeRequest {
	t.Helper()
	stream := awaitStream(t, ctx, js, scraperequestcontract.ScrapeRequestsStreamName)
	consumer, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       durable,
		FilterSubject: scraperequestcontract.ScrapeRequestSubject,
		AckPolicy:     jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatalf("create %s consumer: %v", durable, err)
	}
	msg, err := consumer.Next(jetstream.FetchMaxWait(messageArrivalWait))
	if err != nil {
		t.Fatalf("fetch scrape request for %s: %v", durable, err)
	}
	scrapeRequest, err := scraperequestcontract.UnmarshalScrapeRequest(msg.Data())
	if err != nil {
		t.Fatalf("decode scrape request for %s: %v", durable, err)
	}
	if err := msg.Ack(); err != nil {
		t.Fatalf("ack scrape request for %s: %v", durable, err)
	}
	return scrapeRequest
}
