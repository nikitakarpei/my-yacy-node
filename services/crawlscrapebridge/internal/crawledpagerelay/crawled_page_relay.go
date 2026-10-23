// Package crawledpagerelay asks the scrape service to scrape each page the crawler found
// indexable, so every corpus that reads scraped pages reads the pages the crawler visits.
// A crawled page that cannot be read halts the relay and stays pending for an operator. A
// scrape request that cannot be published returns its crawled page for a later try.
package crawledpagerelay

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type ScrapeRequests interface {
	AskToScrape(ctx context.Context, pageURL canonicalurl.CanonicalURL) bool
}

type CrawledPageRelay struct {
	scrapeRequests ScrapeRequests
	concurrency    int
}

func NewCrawledPageRelay(scrapeRequests ScrapeRequests, concurrency int) *CrawledPageRelay {
	return &CrawledPageRelay{scrapeRequests: scrapeRequests, concurrency: concurrency}
}

func (r *CrawledPageRelay) Run(ctx context.Context, crawledPages pullintake.PendingMessages) error {
	return pullintake.Run(ctx, crawledPages, r.concurrency, r.relay)
}

func (r *CrawledPageRelay) relay(ctx context.Context, crawledPage pullintake.PendingMessage) error {
	page, err := yacycrawlcontract.UnmarshalCrawledPage(crawledPage.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, crawledPage.Identity(), err)
	}
	if !r.scrapeRequests.AskToScrape(ctx, page.PageURL) {
		crawledPage.ReturnAfter(ctx, pullintake.ReturnPause)
		return nil
	}
	crawledPage.Acknowledge(ctx)
	return nil
}
