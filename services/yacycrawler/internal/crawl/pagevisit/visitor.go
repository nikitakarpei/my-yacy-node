// Package pagevisit fetches one URL and hands what it holds to absorption.
package pagevisit

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/reachedpagepublication"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

const msgRecrawlRecordFailed = "recrawl record failed, next visit may be redundant"

type Visitor interface {
	Visit(ctx context.Context, canonicalURL canonicalurl.CanonicalURL) (VisitOutcome, error)
}

type visitor struct {
	fetcher  pagefetch.Fetcher
	recrawl  RecrawlRule
	absorber pageabsorption.Absorber
	observer VisitProgress
	reached  *reachedpagepublication.Publisher
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

	outcome, err := v.fetcher.Fetch(ctx, url.String(), decision.Version)
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
	absorption, err := v.absorber.Absorb(ctx, outcome.Page)
	if err != nil {
		return VisitOutcome{}, err
	}
	v.recordVisit(ctx, url, outcome.Version)
	if absorption.Disposal == disposal.NotDisposed {
		if err := v.reached.Publish(ctx, outcome.Page.FinalURL); err != nil {
			return VisitOutcome{}, err
		}
	}
	return VisitOutcome{
		Conclusion:     VisitCompleted,
		Fetched:        true,
		DiscoveredURLs: absorption.DiscoveredURLs,
		Disposal:       absorption.Disposal,
	}, nil
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
