// Package pagevisit fetches one URL and turns what it holds into the outcome of a page visit.
package pagevisit

import (
	"context"
	"errors"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

type PageVisitor interface {
	VisitPage(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) PageVisitOutcome
}

type pageVisitor struct {
	pageFetcher                PageFetcher
	recrawlRule                RecrawlRule
	visitedPages               VisitedPages
	htmlPageReading            HTMLPageReading
	refusalEnforcementObserver RefusalEnforcementObserver
	crawledPages               CrawledPages
}

//nolint:revive // a page visitor names every collaborator one page visit needs
func New(
	pageFetcher PageFetcher,
	recrawlRule RecrawlRule,
	visitedPages VisitedPages,
	htmlPageReading HTMLPageReading,
	refusalEnforcementObserver RefusalEnforcementObserver,
	crawledPages CrawledPages,
) PageVisitor {
	return &pageVisitor{
		pageFetcher:                pageFetcher,
		recrawlRule:                recrawlRule,
		visitedPages:               visitedPages,
		htmlPageReading:            htmlPageReading,
		refusalEnforcementObserver: refusalEnforcementObserver,
		crawledPages:               crawledPages,
	}
}

func (visitor *pageVisitor) VisitPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
) PageVisitOutcome {
	lastVisit, visited := visitor.visitedPages.LastPageVisitOf(ctx, url)
	if visited && !visitor.recrawlRule.PageDueForRecrawl(lastVisit) {
		return disposedOutcome(disposal.NotDue)
	}
	fetchOutcome := visitor.pageFetcher.Fetch(ctx, url, lastVisit.Version)
	if originAnsweredAboutPage(fetchOutcome.Status) {
		visitor.visitedPages.RecordPageVisit(ctx, url, fetchOutcome.Version)
	}
	return visitor.outcomeOfPageFetch(ctx, url, fetchOutcome)
}

func originAnsweredAboutPage(status pagefetch.FetchStatus) bool {
	return status == pagefetch.FetchSucceeded ||
		status == pagefetch.FetchNotModified ||
		status == pagefetch.FetchRedirected ||
		status == pagefetch.FetchAccessRefused
}

func (visitor *pageVisitor) outcomeOfPageFetch(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	fetchOutcome pagefetch.FetchOutcome,
) PageVisitOutcome {
	switch fetchOutcome.Status {
	case pagefetch.FetchSucceeded:
		return visitor.outcomeOfPageHTML(ctx, url, fetchOutcome.Page)
	case pagefetch.FetchNotModified:
		return disposedOutcome(disposal.NotModified)
	case pagefetch.FetchRedirected:
		return redirectedOutcome(fetchOutcome.RedirectTarget)
	case pagefetch.FetchAccessRefused:
		return disposedOutcome(disposal.AccessRefused)
	case pagefetch.FetchRejected:
		return disposedOutcome(disposal.FetchRejected)
	case pagefetch.FetchRedirectTargetInvalid:
		return disposedOutcome(disposal.RedirectTargetInvalid)
	case pagefetch.FetchOversized:
		return disposedOutcome(disposal.Oversized)
	case pagefetch.FetchDeferred:
		return deferredOutcome(fetchOutcome.DeferFor)
	case pagefetch.FetchFailed:
		return retryableOutcome()
	}
	return retryableOutcome()
}

func (visitor *pageVisitor) outcomeOfPageHTML(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	page pagefetch.FetchedPage,
) PageVisitOutcome {
	reading, err := visitor.htmlPageReading.ReadingOfPage(ctx, url, page)
	if errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
		return disposedOutcome(disposal.UnsupportedMediaType)
	}
	if err != nil {
		return disposedOutcome(disposal.UnreadableHTML)
	}
	if reading.Refusals.RefusesLinkDiscovery {
		visitor.refusalEnforcementObserver.LinkDiscoveryRefusalEnforced(ctx, url)
	}
	visitor.publishCrawledPage(ctx, url, reading.Refusals)
	return crawledOutcome(reading.DiscoveredURLs)
}

func (visitor *pageVisitor) publishCrawledPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	refusals pagerefusals.Refusals,
) {
	if refusals.RefusesIndexing {
		visitor.crawledPages.PublishIndexingRefusedPage(ctx, pageURL)
		return
	}
	visitor.crawledPages.PublishIndexablePage(ctx, pageURL)
}
