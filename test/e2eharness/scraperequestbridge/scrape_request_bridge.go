//go:build e2e

// Package scraperequestbridge relays every indexable crawled page onto the scrape request
// subject, the way a deployment bridges the crawler to the scrape service. It waits for the
// crawler to create the stream it reads from, and it relays until the test ends.
package scraperequestbridge

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	durable            = "scrape_request_bridge"
	streamAppearance   = 60 * time.Second
	consumerAppearance = 60 * time.Second
)

func Relay(t *testing.T, ctx context.Context, natsURL string) {
	t.Helper()
	conn, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect nats at %s: %v", natsURL, err)
	}
	t.Cleanup(conn.Close)
	js, err := jetstream.New(conn)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	awaitCrawledPagesStream(t, ctx, js)
	relaying, err := indexablePages(t, ctx, js).Consume(func(page jetstream.Msg) {
		relayAsScrapeRequest(t, ctx, js, page)
	})
	if err != nil {
		t.Fatalf("relay indexable crawled pages: %v", err)
	}
	t.Cleanup(relaying.Stop)
}

func awaitCrawledPagesStream(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	appeared := pollwait.For(streamAppearance, func() bool {
		_, err := js.Stream(ctx, yacycrawlcontract.CrawledPagesStreamName)
		return err == nil
	})
	if !appeared {
		t.Fatalf(
			"the %s stream did not appear within %s",
			yacycrawlcontract.CrawledPagesStreamName, streamAppearance,
		)
	}
}

func indexablePages(t *testing.T, ctx context.Context, js jetstream.JetStream) jetstream.Consumer {
	t.Helper()
	var pages jetstream.Consumer
	bound := pollwait.For(consumerAppearance, func() bool {
		consumer, err := js.CreateOrUpdateConsumer(
			ctx,
			yacycrawlcontract.CrawledPagesStreamName,
			jetstream.ConsumerConfig{
				Durable:       durable,
				FilterSubject: yacycrawlcontract.IndexablePageSubject,
				DeliverPolicy: jetstream.DeliverAllPolicy,
				AckPolicy:     jetstream.AckExplicitPolicy,
			},
		)
		if err != nil {
			return false
		}
		pages = consumer
		return true
	})
	if !bound {
		t.Fatalf("the %s consumer did not bind within %s", durable, consumerAppearance)
	}
	return pages
}

func relayAsScrapeRequest(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	page jetstream.Msg,
) {
	if _, err := js.Publish(ctx, pagescrapecontract.ScrapeRequestSubject, page.Data()); err != nil {
		t.Errorf("publish a scrape request for a crawled page: %v", err)
		return
	}
	if err := page.Ack(); err != nil {
		t.Errorf("ack a relayed crawled page: %v", err)
	}
}
