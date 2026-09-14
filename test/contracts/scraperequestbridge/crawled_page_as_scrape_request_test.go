package scraperequestbridge_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func TestACrawledPageReadsAsAScrapeRequest(t *testing.T) {
	pageURL := canonicalurltest.CanonicalURLOf(t, "https://example.org/a")

	data, err := yacycrawlcontract.MarshalCrawledPage(
		yacycrawlcontract.CrawledPage{PageURL: pageURL},
	)
	if err != nil {
		t.Fatalf("marshal crawled page: %v", err)
	}
	request, err := pagescrapecontract.UnmarshalScrapeRequest(data)
	if err != nil {
		t.Fatalf("read the crawled page as a scrape request: %v", err)
	}

	if request.PageURL != pageURL {
		t.Fatalf("page url = %q, want %q", request.PageURL, pageURL)
	}
	if request.FetchURL != pageURL {
		t.Fatalf("fetch url = %q, want the page url %q", request.FetchURL, pageURL)
	}
}
