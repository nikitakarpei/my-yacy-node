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
	fetches            PageFetcher
	recrawlRule        RecrawlRule
	visitedPages       VisitedPages
	htmlPageReading    HTMLPageReading
	refusalEnforcement RefusalEnforcementObserver
	crawledPages       CrawledPages
}

//nolint:revive // a page visitor names every collaborator one page visit needs
func New(
	fetches PageFetcher,
	recrawlRule RecrawlRule,
	visitedPages VisitedPages,
	htmlPageReading HTMLPageReading,
	refusalEnforcement RefusalEnforcementObserver,
	crawledPages CrawledPages,
) PageVisitor {
	return &pageVisitor{
		fetches:            fetches,
		recrawlRule:        recrawlRule,
		visitedPages:       visitedPages,
		htmlPageReading:    htmlPageReading,
		refusalEnforcement: refusalEnforcement,
		crawledPages:       crawledPages,
	}
}

func (v *pageVisitor) VisitPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
) (PageVisitOutcome, error) {
	decision, err := v.recrawlRule.RecrawlDecisionFor(ctx, url)
	if err != nil {
		return PageVisitOutcome{}, fmt.Errorf("recrawl decision: %w", err)
	}
	if !decision.Due {
		return terminalOutcome(disposal.NotDue, noDiscoveredURLs), nil
	}
	return v.concludeVisit(ctx, url, v.fetches.Fetch(ctx, url, decision.Version))
}

func (v *pageVisitor) concludeVisit(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	fetchOutcome pagefetch.FetchOutcome,
) (PageVisitOutcome, error) {
	switch fetchOutcome.Status {
	case pagefetch.FetchSucceeded:
		return v.visitFetchedPage(ctx, url, fetchOutcome)
	case pagefetch.FetchNotModified:
		v.recordPageVisit(ctx, url, fetchOutcome.Version)
		return terminalOutcome(disposal.NotModified, noDiscoveredURLs), nil
	case pagefetch.FetchAccessRefused:
		v.recordPageVisit(ctx, url, fetchOutcome.Version)
		return terminalOutcome(disposal.AccessRefused, noDiscoveredURLs), nil
	case pagefetch.FetchRejected:
		return terminalOutcome(disposal.FetchRejected, noDiscoveredURLs), nil
	case pagefetch.FetchLandedURLInvalid:
		return terminalOutcome(disposal.LandedURLInvalid, noDiscoveredURLs), nil
	case pagefetch.FetchOversized:
		return terminalOutcome(disposal.Oversized, noDiscoveredURLs), nil
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

func (v *pageVisitor) visitFetchedPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	fetchOutcome pagefetch.FetchOutcome,
) (PageVisitOutcome, error) {
	page := fetchOutcome.Page
	v.recordPageVisit(ctx, url, fetchOutcome.Version)
	reading, err := v.htmlPageReading.ReadingOfPage(ctx, page)
	if errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
		return terminalOutcome(disposal.UnsupportedMediaType, noDiscoveredURLs), nil
	}
	if err != nil {
		return PageVisitOutcome{}, err
	}
	if reading.Refusals.RefusesLinkDiscovery {
		v.refusalEnforcement.LinkDiscoveryRefusalEnforced(ctx, url)
	}
	v.publishCrawledPage(ctx, page.LandedURL, reading.Refusals)
	return terminalOutcome(disposal.NotDisposed, reading.DiscoveredURLs), nil
}

func (v *pageVisitor) publishCrawledPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	refusals pagerefusals.Refusals,
) {
	if refusals.RefusesIndexing {
		v.crawledPages.PublishIndexingRefusedPage(ctx, pageURL)
		return
	}
	v.crawledPages.PublishIndexablePage(ctx, pageURL)
}

func (v *pageVisitor) recordPageVisit(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	v.visitedPages.RecordPageVisit(ctx, url, version)
}
