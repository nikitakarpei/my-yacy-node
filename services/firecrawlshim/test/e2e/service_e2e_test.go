//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/egressproxy"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/natsjetstream"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/pollwait"
)

const scrapeDeadline = 90 * time.Second

const (
	disposalRecallLimit = 120 * time.Second
	disposalStopBound   = 20 * time.Second
)

func TestScrapeServesCrawledMarkdown(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)

	startCrawler(t, ctx, network.Name)
	startCorpusMarkdown(t, ctx, network.Name)
	awaitRecallPreconditions(t, ctx, connectJetStream(t, natsURL))
	shimURL := startFirecrawlShim(t, ctx, network.Name, defaultRecallLimit)

	var result scrapeResult
	served := pollwait.For(scrapeDeadline, func() bool {
		callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		result, _ = scrape(t, callCtx, shimURL, originURL)
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

func TestScrapeStopsEarlyWhenCrawlingDisposesOfThePage(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	natsURL := natsjetstream.Start(t, ctx, network.Name)
	originURL := startOrigin(t, ctx, network.Name)
	egressproxy.Start(t, ctx, network.Name)

	startCrawler(t, ctx, network.Name)
	startCorpusMarkdown(t, ctx, network.Name)
	awaitRecallPreconditions(t, ctx, connectJetStream(t, natsURL))
	shimURL := startFirecrawlShim(t, ctx, network.Name, disposalRecallLimit)

	missingURL := originURL + "missing-page"

	start := time.Now()
	callCtx, cancel := context.WithTimeout(ctx, disposalRecallLimit)
	defer cancel()
	result, status := scrape(t, callCtx, shimURL, missingURL)
	elapsed := time.Since(start)

	if elapsed >= disposalStopBound {
		t.Errorf(
			"scrape of a disposed page took %s, want it to answer well under the %s recall limit",
			elapsed, disposalRecallLimit,
		)
	}
	if status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", status, http.StatusNotFound)
	}
	if result.Success {
		t.Errorf("success = true, want false for a disposed page")
	}
}
