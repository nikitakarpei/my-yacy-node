// Package pagevisit fetches one URL and hands what it holds to absorption.
package pagevisit

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

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
	due, err := v.recrawl.Due(ctx, url)
	if err != nil {
		return VisitOutcome{}, fmt.Errorf("recrawl decision: %w", err)
	}
	if !due {
		v.observer.PageDisposed(disposal.NotDue)
		return VisitOutcome{Conclusion: VisitCompleted}, nil
	}

	outcome, err := v.fetchPage(ctx, url)
	if err != nil {
		return VisitOutcome{}, err
	}

	switch outcome.Status {
	case FetchSucceeded:
		v.observer.PageFetched()
		return v.absorb(ctx, outcome.Page)
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
	page fetchedpage.Page,
) (VisitOutcome, error) {
	links, err := v.absorber.Absorb(ctx, page)
	if err != nil {
		return VisitOutcome{}, err
	}
	return VisitOutcome{Conclusion: VisitCompleted, DiscoveredURLs: links}, nil
}

func (v *Visitor) fetchPage(
	ctx context.Context,
	rawURL string,
) (FetchOutcome, error) {
	start := v.clock.Now()
	outcome, err := v.fetcher.Fetch(ctx, rawURL)
	if err != nil {
		return FetchOutcome{}, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	v.observer.FetchCompleted(v.clock.Now().Sub(start))
	return outcome, nil
}
