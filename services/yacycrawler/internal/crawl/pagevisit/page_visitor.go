// Package pagevisit visits one page of a crawl order: it fetches the page, reads it,
// publishes the crawled page, and puts visits to the pages it links or redirects to on
// the frontier. It records the visit last, so a visit that delivered only part comes back.
package pagevisit

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
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
	storage          *vault.Vault
	pageFetcher      PageFetcher
	recrawlRule      RecrawlRule
	pageVisitRecords PageVisitRecords
	recordObserver   PageVisitRecordObserver
	htmlPageReading  HTMLPageReading
	crawledPages     CrawledPages
	frontier         Frontier
	linkObserver     LinkAdmissionObserver
}

//nolint:revive // a page visitor names every collaborator one page visit needs
func New(
	storage *vault.Vault,
	pageFetcher PageFetcher,
	recrawlRule RecrawlRule,
	pageVisitRecords PageVisitRecords,
	recordObserver PageVisitRecordObserver,
	htmlPageReading HTMLPageReading,
	crawledPages CrawledPages,
	frontier Frontier,
	linkObserver LinkAdmissionObserver,
) PageVisitor {
	return &pageVisitor{
		storage:          storage,
		pageFetcher:      pageFetcher,
		recrawlRule:      recrawlRule,
		pageVisitRecords: pageVisitRecords,
		recordObserver:   recordObserver,
		htmlPageReading:  htmlPageReading,
		crawledPages:     crawledPages,
		frontier:         frontier,
		linkObserver:     linkObserver,
	}
}

func (visitor *pageVisitor) VisitPage(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) PageVisitOutcome {
	url := pendingPageVisit.URL
	lastVisit, visited := visitor.lastPageVisitOf(ctx, url)
	if visited && !visitor.recrawlRule.PageDueForRecrawl(lastVisit) {
		return droppedOutcome(yacycrawlcontract.DropReasonNotDue)
	}
	visitedPage := visitor.fetchAndRead(ctx, url, lastVisit.Version)
	if visitedPage.isReadable() && !visitor.publishCrawledPage(ctx, url, visitedPage) {
		return returnedOutcome(errCrawledPageUnpublished)
	}
	if err := visitor.putOnFrontier(
		ctx, order, pendingPageVisit.CrawlID, visitedPage.nextPageVisitsFrom(pendingPageVisit),
	); err != nil {
		return returnedOutcome(err)
	}
	if visitedPage.startsRecrawlGrace() {
		visitor.recordPageVisit(ctx, url, visitedPage.fetchOutcome.Version)
	}
	return visitedPage.outcome()
}

func (visitor *pageVisitor) lastPageVisitOf(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
) (LastPageVisit, bool) {
	var lastVisit LastPageVisit
	var visited bool
	if err := visitor.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		lastVisit, visited, err = visitor.pageVisitRecords.LastPageVisitOf(tx, url)
		return err
	}); err != nil {
		visitor.recordObserver.LastPageVisitNotRead(ctx, url, err)
		return LastPageVisit{}, false
	}
	return lastVisit, visited
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
	crawlID crawlid.CrawlID,
	nextPageVisits []pendingpagevisit.PendingPageVisit,
) error {
	admittedPageVisits := make([]pendingpagevisit.PendingPageVisit, 0, len(nextPageVisits))
	for _, nextPageVisit := range nextPageVisits {
		if order.Admits(nextPageVisit.URL, nextPageVisit.Depth) {
			admittedPageVisits = append(admittedPageVisits, nextPageVisit)
		}
	}
	if amountOfUnadmittedLinks := len(
		nextPageVisits,
	) - len(
		admittedPageVisits,
	); amountOfUnadmittedLinks > 0 {
		visitor.linkObserver.LinksUnadmitted(ctx, crawlID, amountOfUnadmittedLinks)
	}
	if err := visitor.storage.Update(ctx, func(tx *vault.Txn) error {
		return visitor.frontier.Owe(tx, admittedPageVisits)
	}); err != nil {
		return fmt.Errorf("put %d page visits on the frontier: %w", len(admittedPageVisits), err)
	}
	return nil
}

func (visitor *pageVisitor) recordPageVisit(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	if err := visitor.storage.Update(ctx, func(tx *vault.Txn) error {
		return visitor.pageVisitRecords.RecordPageVisit(tx, url, version)
	}); err != nil {
		visitor.recordObserver.PageVisitNotRecorded(ctx, url, err)
	}
}
