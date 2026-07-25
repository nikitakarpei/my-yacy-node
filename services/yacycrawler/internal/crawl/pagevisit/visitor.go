// Package pagevisit fetches one URL and hands what it holds to absorption.
package pagevisit

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

const msgRecrawlRecordFailed = "recrawl record failed, next visit may be redundant"

type PageAbsorber interface {
	Absorb(ctx context.Context, page fetchedpage.Page) ([]string, error)
}

type Visitor struct {
	fetcher  Fetcher
	recrawl  RecrawlDecision
	absorber PageAbsorber
	observer VisitProgress
	clock    clock.Clock
}

func New(
	fetcher Fetcher,
	recrawl RecrawlDecision,
	absorber PageAbsorber,
	observer VisitProgress,
	clock clock.Clock,
) *Visitor {
	return &Visitor{
		fetcher:  fetcher,
		recrawl:  recrawl,
		absorber: absorber,
		observer: observer,
		clock:    clock,
	}
}

func (v *Visitor) Visit(ctx context.Context, url string) (VisitOutcome, error) {
	revisit, err := v.recrawl.Revisit(ctx, url)
	if err != nil {
		return VisitOutcome{}, fmt.Errorf("recrawl decision: %w", err)
	}
	if !revisit.Due {
		v.observer.PageDisposed(disposal.NotDue)
		return VisitOutcome{Conclusion: VisitCompleted}, nil
	}

	outcome, err := v.fetchPage(ctx, url, revisit)
	if err != nil {
		return VisitOutcome{}, err
	}

	switch outcome.Status {
	case FetchSucceeded:
		v.observer.PageFetched()
		return v.absorb(ctx, url, outcome)
	case FetchNotModified:
		v.recordVisit(ctx, url, outcome.Validators)
		v.observer.PageDisposed(disposal.NotModified)
		return VisitOutcome{Conclusion: VisitCompleted}, nil
	case FetchCeased:
		v.observer.RefusalHonored(refusal.Cease)
		v.observer.PageDisposed(disposal.Refused)
		return VisitOutcome{Conclusion: VisitCompleted}, nil
	case FetchDeferred:
		return VisitOutcome{Conclusion: VisitDeferred, DeferFor: outcome.DeferFor}, nil
	case FetchNotAPage:
		v.observer.PageFetched()
		v.observer.PageDisposed(disposal.NotAPage)
		return VisitOutcome{Conclusion: VisitCompleted}, nil
	case FetchFailed:
		return VisitOutcome{Conclusion: VisitRetryable}, nil
	default:
		return VisitOutcome{Conclusion: VisitRetryable}, nil
	}
}

func (v *Visitor) absorb(
	ctx context.Context,
	url string,
	outcome FetchOutcome,
) (VisitOutcome, error) {
	links, err := v.absorber.Absorb(ctx, outcome.Page)
	if err != nil {
		return VisitOutcome{}, err
	}
	v.recordVisit(ctx, url, outcome.Validators)
	return VisitOutcome{Conclusion: VisitCompleted, DiscoveredURLs: links}, nil
}

func (v *Visitor) fetchPage(
	ctx context.Context,
	rawURL string,
	validators Revisit,
) (FetchOutcome, error) {
	start := v.clock.Now()
	outcome, err := v.fetcher.Fetch(ctx, rawURL, validators)
	if err != nil {
		return FetchOutcome{}, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	v.observer.FetchCompleted(v.clock.Now().Sub(start))
	return outcome, nil
}

func (v *Visitor) recordVisit(ctx context.Context, url string, validators Revisit) {
	if err := v.recrawl.Visited(ctx, url, validators); err != nil {
		slog.WarnContext(ctx, msgRecrawlRecordFailed,
			slog.String("url", url),
			slog.Any("error", err),
		)
	}
}
