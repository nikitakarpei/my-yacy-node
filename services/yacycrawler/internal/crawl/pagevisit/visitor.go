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
	fetcher         pagefetch.Fetcher
	recrawl         RecrawlRule
	ignoredRefusals pagerefusals.IgnoredRefusals
	readingObserver PageReadingObserver
	scrapeRequests  ScrapeRequests
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
	fetchOutcome, err := v.fetcher.Fetch(ctx, url, decision.Version)
	if err != nil {
		return VisitOutcome{}, fmt.Errorf("fetch %s: %w", url, err)
	}
	return v.concludeVisit(ctx, url, fetchOutcome)
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
	reading, err := pagehtmlreading.ReadingOfPage(ctx, page, v.ignoredRefusals)
	if errors.Is(err, pagehtmlreading.ErrPageNotHTML) {
		v.readingObserver.PageHTMLUnreadable(ctx, page.LandedURL, err)
		return completedOutcome(disposal.UnsupportedMediaType, noDiscoveredURLs), nil
	}
	if err != nil {
		return VisitOutcome{}, err
	}
	v.reportEnforcedRefusals(ctx, url, reading.Refusals)
	if reading.Refusals.RefusesIndexing {
		return completedOutcome(disposal.IndexingRefused, reading.DiscoveredURLs), nil
	}
	if err := v.requestScrape(ctx, page.LandedURL); err != nil {
		return VisitOutcome{}, err
	}
	return completedOutcome(disposal.NotDisposed, reading.DiscoveredURLs), nil
}

func (v *visitor) reportEnforcedRefusals(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	refusals pagerefusals.Refusals,
) {
	if refusals.RefusesIndexing {
		v.readingObserver.IndexingRefusalEnforced(ctx, pageURL)
	}
	if refusals.RefusesLinkDiscovery {
		v.readingObserver.LinkDiscoveryRefusalEnforced(ctx, pageURL)
	}
}

func (v *visitor) requestScrape(
	ctx context.Context,
	landedURL canonicalurl.CanonicalURL,
) error {
	if err := v.scrapeRequests.Publish(ctx, landedURL); err != nil {
		return fmt.Errorf("publish scrape request %s: %w", landedURL, err)
	}
	return nil
}

func (v *visitor) recordVisit(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	_ = v.recrawl.RecordVisit(ctx, url, version)
}
