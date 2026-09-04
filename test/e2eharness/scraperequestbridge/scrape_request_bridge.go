//go:build e2e

// Package scraperequestbridge binds the deployment consumer that turns every indexable
// crawled page into a scrape request. It waits for the crawler to create the stream it
// reads from.
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

func Bind(t *testing.T, ctx context.Context, natsURL string) {
	t.Helper()
	conn, err := nats.Connect(natsURL)
	if err != nil {
		t.Fatalf("connect nats at %s: %v", natsURL, err)
	}
	defer conn.Close()
	js, err := jetstream.New(conn)
	if err != nil {
		t.Fatalf("init jetstream: %v", err)
	}
	awaitCrawledPagesStream(t, ctx, js)
	bound := pollwait.For(consumerAppearance, func() bool {
		_, err := js.CreateOrUpdateConsumer(
			ctx,
			yacycrawlcontract.CrawledPagesStreamName,
			jetstream.ConsumerConfig{
				Durable:        durable,
				FilterSubject:  yacycrawlcontract.IndexablePageSubject,
				DeliverSubject: pagescrapecontract.ScrapeRequestSubject,
				DeliverPolicy:  jetstream.DeliverAllPolicy,
				AckPolicy:      jetstream.AckNonePolicy,
				ReplayPolicy:   jetstream.ReplayInstantPolicy,
			},
		)
		return err == nil
	})
	if !bound {
		t.Fatalf("the %s consumer did not bind within %s", durable, consumerAppearance)
	}
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
