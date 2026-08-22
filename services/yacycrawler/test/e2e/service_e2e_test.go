//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/scraperequeststream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	crawlerv1 "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawler/v1"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawloutcomesclienttest"
)

const (
	orderID         = "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d"
	readPageRPCWait = 10 * time.Second
)

func TestCrawlerPublishesEveryPageAnOrderReachesEndToEnd(t *testing.T) {
	ctx := context.Background()

	js, originURL, _ := startCrawlOfOriginSite(t, ctx)

	scrapeRequest := fetchOneScrapeRequest(t, ctx, js)
	if scrapeRequest.CanonicalURL.String() != originURL {
		t.Errorf(
			"scrape request canonical url = %q, want %q",
			scrapeRequest.CanonicalURL,
			originURL,
		)
	}
}

func TestEveryCorpusConsumesTheSameScrapeRequestEndToEnd(t *testing.T) {
	ctx := context.Background()

	js, originURL, _ := startCrawlOfOriginSite(t, ctx)

	first := fetchScrapeRequestForDurable(t, ctx, js, "corpusmarkdown")
	second := fetchScrapeRequestForDurable(t, ctx, js, "corpustext")

	if first.CanonicalURL.String() != originURL || second.CanonicalURL.String() != originURL {
		t.Errorf("corpora read %q and %q, want both %q",
			first.CanonicalURL, second.CanonicalURL, originURL)
	}
}

func TestReadPageAnswersWhatCrawlingDidWithAScrapeRequestEndToEnd(t *testing.T) {
	ctx := context.Background()

	js, originURL, crawlerAddress := startCrawlOfOriginSite(t, ctx)
	fetchOneScrapeRequest(t, ctx, js)

	client := crawloutcomesclienttest.New(t, crawlerAddress)
	rpcCtx, cancel := context.WithTimeout(ctx, readPageRPCWait)
	defer cancel()
	outcome, err := client.ReadPage(rpcCtx, &crawlerv1.ReadPageRequest{Url: originURL})
	if err != nil {
		t.Fatalf("read page: %v", err)
	}

	if outcome.GetCanonicalUrl() != originURL {
		t.Errorf("canonical url = %q, want %q", outcome.GetCanonicalUrl(), originURL)
	}
	if outcome.GetResolvedUrl() != originURL {
		t.Errorf("resolved url = %q, want %q", outcome.GetResolvedUrl(), originURL)
	}
	if outcome.GetDisposal() != nil {
		t.Errorf("disposal = %v, want nil", outcome.GetDisposal())
	}
}

func startCrawlOfOriginSite(
	t *testing.T,
	ctx context.Context,
) (jetstream.JetStream, string, string) {
	t.Helper()

	network := dockernetwork.New(t, ctx)

	crawlNATSURL := natsjetstream.Start(t, ctx, network.Name)
	scraperequeststream.Provision(t, ctx, crawlNATSURL)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)
	crawlerAddress := startCrawler(t, ctx, network.Name)

	js := connectJetStream(t, crawlNATSURL)
	awaitStream(t, ctx, js, yacycrawlcontract.OrdersStreamName)

	order := yacycrawlcontract.CrawlOrder{
		OrderID: orderID,
		Profile: yacycrawlcontract.CrawlProfile{
			Name:            "default",
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
	return js, originURL, crawlerAddress
}
