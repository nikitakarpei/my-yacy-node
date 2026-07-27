//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	ordersSubject      = "yacy.crawl.orders"
	crawledPageSubject = "yacy.crawl.page.markdown"
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

func provisionCrawlInfrastructure(t *testing.T, ctx context.Context, js jetstream.JetStream) {
	t.Helper()
	if err := yacycrawlcontract.EnsureOrdersStream(ctx, js, yacycrawlcontract.OrdersStreamSpec{
		Subject: ordersSubject,
	}); err != nil {
		t.Fatalf("ensure orders stream: %v", err)
	}
	if err := yacycrawlcontract.EnsureRedirectResolutionBucket(
		ctx, js, yacycrawlcontract.RedirectResolutionBucketSpec{},
	); err != nil {
		t.Fatalf("ensure redirect resolution bucket: %v", err)
	}
	if err := yacycrawlcontract.EnsureDisposedPagesBucket(
		ctx, js, yacycrawlcontract.DisposedPagesBucketSpec{},
	); err != nil {
		t.Fatalf("ensure disposed pages bucket: %v", err)
	}
}
