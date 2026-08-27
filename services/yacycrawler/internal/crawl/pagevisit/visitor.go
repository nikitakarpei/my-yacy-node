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
		return VisitOutcome{Conclusion: VisitCompleted, Disposal: disposal.NotDue}, nil
	}

	outcome, err := v.fetcher.Fetch(ctx, url, decision.Version)
	if err != nil {
		return VisitOutcome{}, fmt.Errorf("fetch %s: %w", url, err)
	}

	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		v.observer.PageFetched()
		return v.absorb(ctx, url, outcome)
	case pagefetch.FetchNotModified:
		v.recordVisit(ctx, url, outcome.Version)
		return VisitOutcome{Conclusion: VisitCompleted, Disposal: disposal.NotModified}, nil
	case pagefetch.FetchCeased:
		v.recordVisit(ctx, url, outcome.Version)
		v.observer.RefusalHonored(refusal.Cease)
		return VisitOutcome{Conclusion: VisitCompleted, Disposal: disposal.Refused}, nil
	case pagefetch.FetchDeferred:
		return VisitOutcome{Conclusion: VisitDeferred, DeferFor: outcome.DeferFor}, nil
	case pagefetch.FetchNotAPage:
		v.observer.PageFetched()
		return VisitOutcome{
			Conclusion: VisitCompleted,
			Fetched:    true,
			Disposal:   disposal.NotAPage,
		}, nil
	case pagefetch.FetchFailed:
		return VisitOutcome{Conclusion: VisitRetryable}, nil
	default:
		return VisitOutcome{}, fmt.Errorf("unknown fetch status %d for %s", outcome.Status, url)
	}
}

func (v *visitor) absorb(
	ctx context.Context,
	url canonicalurl.CanonicalURL,
	outcome pagefetch.FetchOutcome,
) (VisitOutcome, error) {
	absorption := v.absorptionOf(ctx, outcome.Page)
	v.recordVisit(ctx, url, outcome.Version)
	if absorption.Disposal == disposal.NotDisposed {
		if err := v.scrapeRequests.Publish(ctx, outcome.Page.LandedURL); err != nil {
			return VisitOutcome{}, fmt.Errorf(
				"publish scrape request %s: %w", outcome.Page.LandedURL, err,
			)
		}
		v.observer.ScrapeRequestPublished()
	}
	return absorption, nil
}

func (v *visitor) absorptionOf(
	ctx context.Context,
	page pagefetch.FetchedPage,
) VisitOutcome {
	if page.Truncated {
		return absorbedPage(disposal.Oversized, nil)
	}
	markup, err := pagemarkup.MarkupFrom(ctx, page.ContentType, page.Body)
	if err != nil {
		slog.WarnContext(ctx, msgMarkupUnreadable,
			slog.String("url", page.LandedURL.String()),
			slog.Any("error", err),
		)
		return absorbedPage(disposal.UnsupportedMediaType, nil)
	}

	refusals := pagerobots.RefusalsFrom(markup)
	discoveredURLs := discoveredURLsFrom(ctx, page, markup, refusals)
	if v.indexingRefusal == Honored && refusesIndexing(page, refusals) {
		return absorbedPage(disposal.IndexingRefused, discoveredURLs)
	}
	return absorbedPage(disposal.NotDisposed, discoveredURLs)
}

func absorbedPage(reason disposal.Reason, discoveredURLs []canonicalurl.CanonicalURL) VisitOutcome {
	return VisitOutcome{
		Conclusion:     VisitCompleted,
		Fetched:        true,
		DiscoveredURLs: discoveredURLs,
		Disposal:       reason,
	}
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
