// Package pagevisit fetches one URL and turns what it holds into the outcome of a visit.
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

type Visitor interface {
	Visit(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) (VisitOutcome, error)
}

type visitor struct {
	fetches            PageFetcher
	recrawl            RecrawlRule
	htmlPageReading    HTMLPageReading
	refusalEnforcement RefusalEnforcementObserver
	crawledPages       CrawledPages
}

func New(
	fetches PageFetcher,
	recrawl RecrawlRule,
	htmlPageReading HTMLPageReading,
	refusalEnforcement RefusalEnforcementObserver,
	crawledPages CrawledPages,
) Visitor {
	return &visitor{
		fetches:            fetches,
		recrawl:            recrawl,
		htmlPageReading:    htmlPageReading,
		refusalEnforcement: refusalEnforcement,
		crawledPages:       crawledPages,
	}
}

func (v *visitor) Visit(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
) (VisitOutcome, error) {
	decision, err := v.recrawl.DecisionFor(ctx, url)
	if err != nil {
		return VisitOutcome{}, fmt.Errorf("recrawl decision: %w", err)
	}
	if !decision.Due {
		return completedOutcome(disposal.NotDue, noDiscoveredURLs), nil
	}
	return v.concludeVisit(ctx, url, v.fetches.Fetch(ctx, url, decision.Version))
}

func (v *visitor) concludeVisit(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	fetchOutcome pagefetch.FetchOutcome,
) (VisitOutcome, error) {
	switch fetchOutcome.Status {
	case pagefetch.FetchSucceeded:
		return v.visitFetchedPage(ctx, url, fetchOutcome)
	case pagefetch.FetchNotModified:
		v.recordVisit(ctx, url, fetchOutcome.Version)
		return completedOutcome(disposal.NotModified, noDiscoveredURLs), nil
	case pagefetch.FetchAccessRefused:
		v.recordVisit(ctx, url, fetchOutcome.Version)
		return completedOutcome(disposal.AccessRefused, noDiscoveredURLs), nil
	case pagefetch.FetchRejected:
		return completedOutcome(disposal.FetchRejected, noDiscoveredURLs), nil
	case pagefetch.FetchLandedURLInvalid:
		return completedOutcome(disposal.LandedURLInvalid, noDiscoveredURLs), nil
	case pagefetch.FetchOversized:
		return completedOutcome(disposal.Oversized, noDiscoveredURLs), nil
	case pagefetch.FetchDeferred:
		return deferredOutcome(fetchOutcome.DeferFor), nil
	case pagefetch.FetchFailed:
		return retryableOutcome(), nil
	default:
		return VisitOutcome{}, fmt.Errorf(
			"unknown fetch status %d for %s",
			fetchOutcome.Status,
			url,
		)
	}
}

func (v *visitor) visitFetchedPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	fetchOutcome pagefetch.FetchOutcome,
) (VisitOutcome, error) {
	page := fetchOutcome.Page
	v.recordVisit(ctx, url, fetchOutcome.Version)
	reading, err := v.htmlPageReading.ReadingOfPage(ctx, page)
	if errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
		return completedOutcome(disposal.UnsupportedMediaType, noDiscoveredURLs), nil
	}
	if err != nil {
		return VisitOutcome{}, err
	}
	if reading.Refusals.RefusesLinkDiscovery {
		v.refusalEnforcement.LinkDiscoveryRefusalEnforced(ctx, url)
	}
	v.reportCrawledPage(ctx, page.LandedURL, reading.Refusals)
	return completedOutcome(disposal.NotDisposed, reading.DiscoveredURLs), nil
}

func (v *visitor) reportCrawledPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	refusals pagerefusals.Refusals,
) {
	if refusals.RefusesIndexing {
		v.crawledPages.ReportIndexingRefusedPage(ctx, pageURL)
		return
	}
	v.crawledPages.ReportIndexablePage(ctx, pageURL)
}

func (v *visitor) recordVisit(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	_ = v.recrawl.RecordVisit(ctx, url, version)
}
