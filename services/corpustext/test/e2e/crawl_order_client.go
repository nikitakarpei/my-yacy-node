//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	ordersSubject              = "yacy.crawl.orders"
	ordersStreamAppearanceWait = 60 * time.Second
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

func awaitOrdersStream(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	appeared := pollwait.For(ordersStreamAppearanceWait, func() bool {
		_, err := js.Stream(ctx, yacycrawlcontract.OrdersStreamName)
		return err == nil
	})
	if !appeared {
		t.Fatalf("orders stream did not appear within %s", ordersStreamAppearanceWait)
	}
}

func publishCrawlOrder(
	t *testing.T,
	ctx context.Context,
	js jetstream.JetStream,
	originURL string,
) {
	t.Helper()
	order := yacycrawlcontract.CrawlOrder{
		OrderID: "b3f2a1c0-4d5e-4f6a-8b9c-0d1e2f3a4b5c",
		Profile: yacycrawlcontract.CrawlProfile{
			Name:            "default",
			Scope:           yacycrawlcontract.ScopeDomain,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxDepth:        0,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
		SeedURLs: []string{originURL},
	}
	data, err := yacycrawlcontract.MarshalCrawlOrder(order)
	if err != nil {
		t.Fatalf("marshal order: %v", err)
	}
	if _, err := js.Publish(ctx, ordersSubject, data); err != nil {
		t.Fatalf("publish order: %v", err)
	}
}
