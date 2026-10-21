// Package pagevisit visits one page of a crawl order: it fetches the page, reads it,
// publishes the crawled page, and puts visits to the pages it links or redirects to on
// the frontier. It records the visit last, so a visit that delivered only part comes back.
package pagevisit

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

var errCrawledPageUnpublished = errors.New("crawled page not published")

type PageVisitor interface {
	VisitPage(
		ctx context.Context,
		order acceptedorder.AcceptedOrder,
		pendingPageVisit pendingpagevisit.PendingPageVisit,
	) PageVisitOutcome
}

type pageVisitor struct {
	pageFetcher      PageFetcher
	recrawlRule      RecrawlRule
	pageVisitRecords PageVisitRecords
	htmlPageReading  HTMLPageReading
	crawledPages     CrawledPages
	frontier         Frontier
}

//nolint:revive // a page visitor names every collaborator one page visit needs
func New(
	pageFetcher PageFetcher,
	recrawlRule RecrawlRule,
	pageVisitRecords PageVisitRecords,
	htmlPageReading HTMLPageReading,
	crawledPages CrawledPages,
	frontier Frontier,
) PageVisitor {
	return &pageVisitor{
		pageFetcher:      pageFetcher,
		recrawlRule:      recrawlRule,
		pageVisitRecords: pageVisitRecords,
		htmlPageReading:  htmlPageReading,
		crawledPages:     crawledPages,
		frontier:         frontier,
	}
}

func (visitor *pageVisitor) VisitPage(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) PageVisitOutcome {
	url := pendingPageVisit.URL
	lastVisit, visited := visitor.pageVisitRecords.LastPageVisitOf(ctx, url)
	if visited && !visitor.recrawlRule.PageDueForRecrawl(lastVisit) {
		return droppedOutcome(pagevisitdrop.NotDue)
	}
	visitedPage := visitor.fetchAndRead(ctx, url, lastVisit.Version)
	if visitedPage.isReadable() && !visitor.publishCrawledPage(ctx, url, visitedPage) {
		return returnedOutcome(errCrawledPageUnpublished)
	}
	if err := visitor.putOnFrontier(
		ctx, order, visitedPage.nextPageVisitsFrom(pendingPageVisit),
	); err != nil {
		return returnedOutcome(err)
	}
	if visitedPage.startsRecrawlGrace() {
		visitor.pageVisitRecords.RecordPageVisit(ctx, url, visitedPage.fetchOutcome.Version)
	}
	return visitedPage.outcome()
}

func (visitor *pageVisitor) fetchAndRead(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) visitedPage {
	fetchOutcome := visitor.pageFetcher.Fetch(ctx, url, knownVersion)
	if fetchOutcome.Status != pagefetch.FetchSucceeded {
		return visitedPage{fetchOutcome: fetchOutcome}
	}
	reading, err := visitor.htmlPageReading.ReadingOfPage(ctx, url, fetchOutcome.Page)
	return visitedPage{fetchOutcome: fetchOutcome, reading: reading, readingErr: err}
}

func (visitor *pageVisitor) publishCrawledPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	page visitedPage,
) bool {
	if page.reading.Refusals.RefusesIndexing {
		return visitor.crawledPages.PublishIndexingRefusedPage(ctx, url)
	}
	return visitor.crawledPages.PublishIndexablePage(ctx, url)
}

func (visitor *pageVisitor) putOnFrontier(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
	nextPageVisits []pendingpagevisit.PendingPageVisit,
) error {
	admittedPageVisits := make([]pendingpagevisit.PendingPageVisit, 0, len(nextPageVisits))
	for _, nextPageVisit := range nextPageVisits {
		if order.Admits(nextPageVisit.URL, nextPageVisit.Depth) {
			admittedPageVisits = append(admittedPageVisits, nextPageVisit)
		}
	}
	return visitor.frontier.Publish(ctx, admittedPageVisits)
}
