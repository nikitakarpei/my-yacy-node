//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/scraperequestbridge"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/scraperequeststream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const orderID = "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"

func TestCrawlerPublishesEveryPageAnOrderReachesEndToEnd(t *testing.T) {
	ctx := context.Background()

	js, originURL := startCrawlOfOriginSite(t, ctx)

	scrapeRequest := fetchOneScrapeRequest(t, ctx, js)
	if scrapeRequest.PageURL.String() != originURL {
		t.Errorf(
			"scrape request page url = %q, want %q",
			scrapeRequest.PageURL,
			originURL,
		)
	}
}

func TestTwoCrawlersVisitEachURLOnceEndToEnd(t *testing.T) {
	ctx := context.Background()

	js, originURL := startCrawlOfOriginSiteAcross(t, ctx, 2)

	scrapeRequests := fetchEveryScrapeRequest(t, ctx, js)

	if len(scrapeRequests) != 1 {
		t.Fatalf(
			"two crawlers published %d scrape requests for one url, want 1",
			len(scrapeRequests),
		)
	}
	if scrapeRequests[0].PageURL.String() != originURL {
		t.Errorf("scrape request page url = %q, want %q", scrapeRequests[0].PageURL, originURL)
	}
}

func startCrawlOfOriginSite(
	t *testing.T,
	ctx context.Context,
) (jetstream.JetStream, string) {
	t.Helper()
	return startCrawlOfOriginSiteAcross(t, ctx, 1)
}

func startCrawlOfOriginSiteAcross(
	t *testing.T,
	ctx context.Context,
	crawlers int,
) (jetstream.JetStream, string) {
	t.Helper()

	network := dockernetwork.New(t, ctx)

	crawlNATSURL := natsjetstream.Start(t, ctx, network.Name)
	scraperequeststream.Provision(t, ctx, crawlNATSURL)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	startCrawlers(t, ctx, network.Name, crawlers)
	scraperequestbridge.Bind(t, ctx, crawlNATSURL)

	js := connectJetStream(t, crawlNATSURL)
	awaitStream(t, ctx, js, yacycrawlcontract.OrdersStreamName)

	order := yacycrawlcontract.CrawlOrder{
		OrderID: orderID,
		Profile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.ScopeDomain,
			URLMustMatch:    yacycrawlcontract.MatchAll,
			MaxDepth:        0,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
		SeedURLs: []canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, originURL)},
	}
	data, err := yacycrawlcontract.MarshalCrawlOrder(order)
	if err != nil {
		t.Fatalf("marshal order: %v", err)
	}
	if _, err := js.Publish(ctx, ordersSubject, data); err != nil {
		t.Fatalf("publish order: %v", err)
	}
	return js, originURL
}
