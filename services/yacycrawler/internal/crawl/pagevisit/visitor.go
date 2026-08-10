// Package pagevisit fetches one URL and hands what it holds to absorption.
package pagevisit

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

const msgRecrawlRecordFailed = "recrawl record failed, next visit may be redundant"

type Visitor interface {
	Visit(ctx context.Context, url string) (VisitOutcome, error)
}

type visitor struct {
	fetcher  Fetcher
	recrawl  RecrawlRule
	absorber pageabsorption.Absorber
	observer VisitProgress
	clock    clock.Clock
}

func (v *visitor) Visit(ctx context.Context, url string) (VisitOutcome, error) {
	decision, err := v.recrawl.DecisionFor(ctx, url)
	if err != nil {
		return VisitOutcome{}, fmt.Errorf("recrawl decision: %w", err)
	}
	if !decision.Due {
		return VisitOutcome{Conclusion: VisitCompleted, Disposal: disposal.NotDue}, nil
	}

	outcome, err := v.fetchPage(ctx, url, decision.Version)
	if err != nil {
		return VisitOutcome{}, err
	}

	switch outcome.Status {
	case FetchSucceeded:
		v.observer.PageFetched()
		return v.absorb(ctx, url, outcome)
	case FetchNotModified:
		v.recordVisit(ctx, url, outcome.Version)
		return VisitOutcome{Conclusion: VisitCompleted, Disposal: disposal.NotModified}, nil
	case FetchCeased:
		v.recordVisit(ctx, url, outcome.Version)
		v.observer.RefusalHonored(refusal.Cease)
		return VisitOutcome{Conclusion: VisitCompleted, Disposal: disposal.Refused}, nil
	case FetchDeferred:
		return VisitOutcome{Conclusion: VisitDeferred, DeferFor: outcome.DeferFor}, nil
	case FetchNotAPage:
		v.observer.PageFetched()
		return VisitOutcome{
			Conclusion: VisitCompleted,
			Fetched:    true,
			Disposal:   disposal.NotAPage,
		}, nil
	case FetchFailed:
		return VisitOutcome{Conclusion: VisitRetryable}, nil
	default:
		return VisitOutcome{}, fmt.Errorf("unknown fetch status %d for %s", outcome.Status, url)
	}
}

func (v *visitor) fetchPage(
	ctx context.Context,
	rawURL string,
	knownVersion PageVersion,
) (FetchOutcome, error) {
	start := v.clock.Now()
	outcome, err := v.fetcher.Fetch(ctx, rawURL, knownVersion)
	if err != nil {
		return FetchOutcome{}, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	v.observer.FetchCompleted(v.clock.Now().Sub(start))
	return outcome, nil
}

func (v *visitor) absorb(
	ctx context.Context,
	url string,
	outcome FetchOutcome,
) (VisitOutcome, error) {
	absorption, err := v.absorber.Absorb(ctx, outcome.Page)
	if err != nil {
		return VisitOutcome{}, err
	}
	v.recordVisit(ctx, url, outcome.Version)
	return VisitOutcome{
		Conclusion:     VisitCompleted,
		Fetched:        true,
		DiscoveredURLs: absorption.DiscoveredURLs,
		Disposal:       absorption.Disposal,
	}, nil
}

func (v *visitor) recordVisit(ctx context.Context, url string, version PageVersion) {
	if err := v.recrawl.RecordVisit(ctx, url, version); err != nil {
		slog.WarnContext(ctx, msgRecrawlRecordFailed,
			slog.String("url", url),
			slog.Any("error", err),
		)
	}
}
