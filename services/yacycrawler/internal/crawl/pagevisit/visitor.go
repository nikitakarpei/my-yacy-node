// Package pagevisit fetches one URL and turns what it holds into the outcome of a visit.
package pagevisit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagelinks"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

const (
	msgRecrawlRecordFailed = "recrawl record failed, next visit may be redundant"
	msgLinkReadingFailed   = "page links unreadable"
)

type Visitor interface {
	Visit(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) (VisitOutcome, error)
}

type visitor struct {
	fetcher         pagefetch.Fetcher
	recrawl         RecrawlRule
	pageLinks       PageLinksSource
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
		if err := v.scrapeRequests.Publish(ctx, outcome.Page.FinalURL); err != nil {
			return VisitOutcome{}, fmt.Errorf(
				"publish scrape request %s: %w", outcome.Page.FinalURL, err,
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
	links, err := v.pageLinks(ctx, page.FinalURL.String(), page.ContentType, page.Body)
	if err != nil {
		slog.WarnContext(ctx, msgLinkReadingFailed,
			slog.String("url", page.FinalURL.String()),
			slog.Any("error", err),
		)
		return absorbedPage(disposalOfUnreadableLinks(err), nil)
	}

	if v.indexingRefusal == Honored && (links.RefusesIndexing || page.RefusesIndexing) {
		return absorbedPage(disposal.IndexingRefused, discoveredURLsOf(page, links))
	}
	return absorbedPage(disposal.NotDisposed, discoveredURLsOf(page, links))
}

func absorbedPage(reason disposal.Reason, discoveredURLs []canonicalurl.CanonicalURL) VisitOutcome {
	return VisitOutcome{
		Conclusion:     VisitCompleted,
		Fetched:        true,
		DiscoveredURLs: discoveredURLs,
		Disposal:       reason,
	}
}

func disposalOfUnreadableLinks(err error) disposal.Reason {
	if errors.Is(err, pagelinks.ErrNotHTML) {
		return disposal.UnsupportedMediaType
	}
	return disposal.Unextractable
}

func discoveredURLsOf(
	page pagefetch.FetchedPage,
	links pagelinks.PageLinks,
) []canonicalurl.CanonicalURL {
	if links.RefusesLinkDiscovery || page.RefusesLinkDiscovery {
		return nil
	}
	return links.LinkedURLs
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
