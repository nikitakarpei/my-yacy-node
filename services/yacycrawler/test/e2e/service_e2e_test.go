//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
)

func TestCrawlerIsOrderDrivenEndToEnd(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	crawlNATSURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	startCrawler(t, ctx, network.Name)

	js := connectJetStream(t, crawlNATSURL)
	awaitOrdersStream(t, ctx, js)

	order := yacycrawlcontract.CrawlOrder{
		OrderID: "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
		Profile: yacycrawlcontract.CrawlProfile{
			Name:            "default",
			Scope:           yacycrawlcontract.ScopeDomain,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxDepth:        0,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
		SeedURLs: []yacycrawlcontract.CanonicalURL{canonicalurltest.CanonicalURLOf(t, originURL)},
	}
	data, err := yacycrawlcontract.MarshalCrawlOrder(order)
	if err != nil {
		t.Fatalf("marshal order: %v", err)
	}
	if _, err := js.Publish(ctx, ordersSubject, data); err != nil {
		t.Fatalf("publish order: %v", err)
	}

	representation := fetchOnePageRWIRepresentation(t, ctx, js)
	if representation.CanonicalURL != canonicalurltest.CanonicalURLOf(t, originURL) {
		t.Errorf("representation canonical url = %q, want %q",
			representation.CanonicalURL, originURL)
	}
	if len(representation.Postings) == 0 {
		t.Error("representation carries no postings")
	}
}
