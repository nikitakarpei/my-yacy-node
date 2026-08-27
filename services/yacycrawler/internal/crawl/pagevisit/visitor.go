// Package pagevisit fetches one URL and turns what it holds into the outcome of a visit.
package pagevisit

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/linkdiscovery"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagemarkup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerobots"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

const (
	msgRecrawlRecordFailed = "recrawl record failed, next visit may be redundant"
	msgMarkupUnreadable    = "page markup unreadable"
)

type Visitor interface {
	Visit(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) (VisitOutcome, error)
}

type visitor struct {
	fetcher         pagefetch.Fetcher
	recrawl         RecrawlRule
	ignoredRefusals IgnoredRefusals
	progress        VisitProgress
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
	case pagefetch.FetchCeased:
		v.recordVisit(ctx, url, fetchOutcome.Version)
		v.progress.RefusalHonored(refusal.Cease)
		return completedOutcome(disposal.CrawlCeased, noDiscoveredURLs), nil
	case pagefetch.FetchRejected:
		v.progress.PageFetched()
		return completedOutcome(disposal.FetchRejected, noDiscoveredURLs), nil
	case pagefetch.FetchLandedURLInvalid:
		v.progress.PageFetched()
		return completedOutcome(disposal.LandedURLInvalid, noDiscoveredURLs), nil
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
	v.progress.PageFetched()
	v.recordVisit(ctx, url, fetchOutcome.Version)
	if page.Truncated {
		return completedOutcome(disposal.Oversized, noDiscoveredURLs), nil
	}
	markup, readable := markupOfPage(ctx, page)
	if !readable {
		return completedOutcome(disposal.UnsupportedMediaType, noDiscoveredURLs), nil
	}
	refusals := refusalsHonoredBy(
		pagerobots.RefusalsOfPage(page.RobotsDirectives, markup),
		v.ignoredRefusals,
	)
	discoveredURLs := discoveredURLsFrom(ctx, markup, page.LandedURL, refusals)
	if refusals.RefusesIndexing {
		return completedOutcome(disposal.IndexingRefused, discoveredURLs), nil
	}
	if err := v.requestScrape(ctx, page.LandedURL); err != nil {
		return VisitOutcome{}, err
	}
	return completedOutcome(disposal.NotDisposed, discoveredURLs), nil
}

func markupOfPage(
	ctx context.Context,
	page pagefetch.FetchedPage,
) (pagemarkup.Markup, bool) {
	markup, err := pagemarkup.MarkupFrom(ctx, page.ContentType, page.Body)
	if err != nil {
		slog.WarnContext(ctx, msgMarkupUnreadable,
			slog.String("url", page.LandedURL.String()),
			slog.Any("error", err),
		)
		return pagemarkup.Markup{}, false
	}
	return markup, true
}

func refusalsHonoredBy(
	stated pagerobots.Refusals,
	ignored IgnoredRefusals,
) pagerobots.Refusals {
	return pagerobots.Refusals{
		RefusesIndexing:      stated.RefusesIndexing && !ignored.IndexingRefusal,
		RefusesLinkDiscovery: stated.RefusesLinkDiscovery,
	}
}

func discoveredURLsFrom(
	ctx context.Context,
	markup pagemarkup.Markup,
	landedURL canonicalurl.CanonicalURL,
	refusals pagerobots.Refusals,
) []canonicalurl.CanonicalURL {
	if refusals.RefusesLinkDiscovery {
		return nil
	}
	return linkdiscovery.LinkedURLsFrom(ctx, markup, landedURL)
}

func (v *visitor) requestScrape(
	ctx context.Context,
	landedURL canonicalurl.CanonicalURL,
) error {
	if err := v.scrapeRequests.Publish(ctx, landedURL); err != nil {
		return fmt.Errorf("publish scrape request %s: %w", landedURL, err)
	}
	v.progress.ScrapeRequestPublished()
	return nil
}

func (v *visitor) recordVisit(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) {
	if err := v.recrawl.RecordVisit(ctx, url, version); err != nil {
		slog.WarnContext(ctx, msgRecrawlRecordFailed,
			slog.String("url", url.String()),
			slog.Any("error", err),
		)
	}
}
