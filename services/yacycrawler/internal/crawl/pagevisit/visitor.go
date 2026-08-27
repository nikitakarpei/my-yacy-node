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
	indexingRefusal IndexingRefusal
	observer        VisitProgress
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
		return completedVisit(disposal.NotDue, nil), nil
	}
	fetch, err := v.fetcher.Fetch(ctx, url, decision.Version)
	if err != nil {
		return VisitOutcome{}, fmt.Errorf("fetch %s: %w", url, err)
	}
	return v.outcomeOfFetch(ctx, url, fetch)
}

func (v *visitor) outcomeOfFetch(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	fetch pagefetch.FetchOutcome,
) (VisitOutcome, error) {
	switch fetch.Status {
	case pagefetch.FetchSucceeded:
		return v.outcomeOfFetchedPage(ctx, url, fetch)
	case pagefetch.FetchNotModified:
		return v.outcomeOfUnmodifiedPage(ctx, url, fetch.Version), nil
	case pagefetch.FetchCeased:
		return v.outcomeOfCeasedFetch(ctx, url, fetch.Version), nil
	case pagefetch.FetchNotAPage:
		return v.outcomeOfNonPage(), nil
	case pagefetch.FetchDeferred:
		return deferredVisit(fetch.DeferFor), nil
	case pagefetch.FetchFailed:
		return retryableVisit(), nil
	default:
		return VisitOutcome{}, fmt.Errorf("unknown fetch status %d for %s", fetch.Status, url)
	}
}

func (v *visitor) outcomeOfFetchedPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	fetch pagefetch.FetchOutcome,
) (VisitOutcome, error) {
	v.observer.PageFetched()
	outcome := v.outcomeOfPageContent(ctx, fetch.Page)
	v.recordVisit(ctx, url, fetch.Version)
	if outcome.Disposal != disposal.NotDisposed {
		return outcome, nil
	}
	if err := v.requestScrape(ctx, fetch.Page.LandedURL); err != nil {
		return VisitOutcome{}, err
	}
	return outcome, nil
}

func (v *visitor) outcomeOfPageContent(
	ctx context.Context,
	page pagefetch.FetchedPage,
) VisitOutcome {
	if page.Truncated {
		return completedVisit(disposal.Oversized, nil)
	}
	markup, err := pagemarkup.MarkupFrom(ctx, page.ContentType, page.Body)
	if err != nil {
		slog.WarnContext(ctx, msgMarkupUnreadable,
			slog.String("url", page.LandedURL.String()),
			slog.Any("error", err),
		)
		return completedVisit(disposal.UnsupportedMediaType, nil)
	}
	refusals := pagerobots.RefusalsFrom(markup)
	discoveredURLs := discoveredURLsFrom(ctx, page, markup, refusals)
	if v.indexingRefusal == Honored && refusesIndexing(page, refusals) {
		return completedVisit(disposal.IndexingRefused, discoveredURLs)
	}
	return completedVisit(disposal.NotDisposed, discoveredURLs)
}

func discoveredURLsFrom(
	ctx context.Context,
	page pagefetch.FetchedPage,
	markup pagemarkup.Markup,
	refusals pagerobots.Refusals,
) []canonicalurl.CanonicalURL {
	if page.RefusesLinkDiscovery || refusals.RefusesLinkDiscovery {
		return nil
	}
	return linkdiscovery.LinkedURLsFrom(ctx, markup, page.LandedURL)
}

func refusesIndexing(page pagefetch.FetchedPage, refusals pagerobots.Refusals) bool {
	return page.RefusesIndexing || refusals.RefusesIndexing
}

func (v *visitor) requestScrape(
	ctx context.Context,
	landedURL canonicalurl.CanonicalURL,
) error {
	if err := v.scrapeRequests.Publish(ctx, landedURL); err != nil {
		return fmt.Errorf("publish scrape request %s: %w", landedURL, err)
	}
	v.observer.ScrapeRequestPublished()
	return nil
}

func (v *visitor) outcomeOfUnmodifiedPage(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) VisitOutcome {
	v.recordVisit(ctx, url, version)
	return completedVisit(disposal.NotModified, nil)
}

func (v *visitor) outcomeOfCeasedFetch(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	version pagefetch.PageVersion,
) VisitOutcome {
	v.recordVisit(ctx, url, version)
	v.observer.RefusalHonored(refusal.Cease)
	return completedVisit(disposal.Refused, nil)
}

func (v *visitor) outcomeOfNonPage() VisitOutcome {
	v.observer.PageFetched()
	return completedVisit(disposal.NotAPage, nil)
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
