//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

const scrapeDeadline = 90 * time.Second

func TestScrapeServesCrawledMarkdown(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)

	js := connectJetStream(t, natsURL)
	provisionCrawlInfrastructure(t, ctx, js)

	startCrawler(t, ctx, network.Name)
	startCorpusMarkdown(t, ctx, network.Name)
	startCorpusRecall(t, ctx, network.Name)
	shimURL := startFirecrawlShim(t, ctx, network.Name)

	var result scrapeResult
	served := pollwait.For(scrapeDeadline, func() bool {
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		result = scrape(t, callCtx, shimURL, originURL)
		return result.Success && result.Data.Markdown != ""
	})
	if !served {
		t.Fatalf("scrape never returned markdown within %s", scrapeDeadline)
	}

	if !strings.Contains(result.Data.Markdown, originBody) {
		t.Errorf(
			"scraped markdown = %q, want it to contain %q",
			result.Data.Markdown,
			originBody,
		)
	}
	if result.Data.Metadata.SourceURL == "" {
		t.Errorf("sourceURL is empty, want the crawled page's canonical url")
	}
}
