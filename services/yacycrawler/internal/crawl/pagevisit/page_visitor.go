// Package pagevisit fetches one URL and turns what it holds into the outcome of a page visit.
package pagevisit

import (
	"context"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

type PageVisitor interface {
	VisitPage(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) (PageVisitOutcome, error)
}

type pageVisitor struct {
	pageFetcher                PageFetcher
	recrawlRule                RecrawlRule
	visitedPages               VisitedPages
	htmlPageReading            HTMLPageReading
	refusalEnforcementObserver RefusalEnforcementObserver
	crawledPages               CrawledPages
	pageVisitFailureObserver   PageVisitFailureObserver
}

//nolint:revive // a page visitor names every collaborator one page visit needs
func New(
	pageFetcher PageFetcher,
	recrawlRule RecrawlRule,
	visitedPages VisitedPages,
	htmlPageReading HTMLPageReading,
	refusalEnforcementObserver RefusalEnforcementObserver,
	crawledPages CrawledPages,
	pageVisitFailureObserver PageVisitFailureObserver,
) PageVisitor {
	return &pageVisitor{
		pageFetcher:                pageFetcher,
		recrawlRule:                recrawlRule,
		visitedPages:               visitedPages,
		htmlPageReading:            htmlPageReading,
		refusalEnforcementObserver: refusalEnforcementObserver,
		crawledPages:               crawledPages,
		pageVisitFailureObserver:   pageVisitFailureObserver,
	}
}

func (visitor *pageVisitor) VisitPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
) (PageVisitOutcome, error) {
	lastVisit, visited, err := visitor.visitedPages.LastPageVisitOf(ctx, url)
	if err != nil {
		visitor.pageVisitFailureObserver.LastPageVisitUnreadable(ctx, url, err)
		return retryableOutcome(), nil
	}
	if visited && !visitor.recrawlRule.PageDueForRecrawl(lastVisit) {
		return disposedOutcome(disposal.NotDue), nil
	}
	fetchOutcome := visitor.pageFetcher.Fetch(ctx, url, lastVisit.Version)
	if originAnsweredAboutPage(fetchOutcome.Status) {
		visitor.visitedPages.RecordPageVisit(ctx, url, fetchOutcome.Version)
	}
	if fetchOutcome.Status == pagefetch.FetchSucceeded {
		return visitor.outcomeOfFetchedPage(ctx, url, fetchOutcome.Page), nil
	}
	return outcomeOfUnfetchedPage(fetchOutcome, url)
}

func originAnsweredAboutPage(status pagefetch.FetchStatus) bool {
	return status == pagefetch.FetchSucceeded ||
		status == pagefetch.FetchNotModified ||
		status == pagefetch.FetchAccessRefused
}

func (visitor *pageVisitor) outcomeOfFetchedPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	page pagefetch.FetchedPage,
) PageVisitOutcome {
	reading, err := visitor.htmlPageReading.ReadingOfPage(ctx, page)
	if errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
		return disposedOutcome(disposal.UnsupportedMediaType)
	}
	if err != nil {
		visitor.pageVisitFailureObserver.PageHTMLUnreadable(ctx, url, err)
		return retryableOutcome()
	}
	if reading.Refusals.RefusesLinkDiscovery {
		visitor.refusalEnforcementObserver.LinkDiscoveryRefusalEnforced(ctx, url)
	}
	visitor.publishCrawledPage(ctx, page.LandedURL, reading.Refusals)
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

func outcomeOfUnfetchedPage(
	fetchOutcome pagefetch.FetchOutcome,
	url canonicalurl.CanonicalURL,
) (PageVisitOutcome, error) {
	switch fetchOutcome.Status {
	case pagefetch.FetchNotModified:
		return disposedOutcome(disposal.NotModified), nil
	case pagefetch.FetchAccessRefused:
		return disposedOutcome(disposal.AccessRefused), nil
	case pagefetch.FetchRejected:
		return disposedOutcome(disposal.FetchRejected), nil
	case pagefetch.FetchLandedURLInvalid:
		return disposedOutcome(disposal.LandedURLInvalid), nil
	case pagefetch.FetchOversized:
		return disposedOutcome(disposal.Oversized), nil
	case pagefetch.FetchDeferred:
		return deferredOutcome(fetchOutcome.DeferFor), nil
	case pagefetch.FetchFailed:
		return retryableOutcome(), nil
	default:
		return PageVisitOutcome{}, fmt.Errorf(
			"unknown fetch status %d for %s",
			fetchOutcome.Status,
			url,
		)
	}
}
