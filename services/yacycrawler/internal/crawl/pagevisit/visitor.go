// Package pagevisit fetches one URL and hands what it holds to absorption.
package pagevisit

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/fetchedpage"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
)

const (
	msgRecrawlRecordFailed      = "recrawl record failed, next visit may be redundant"
	msgDisposedPageRecordFailed = "disposed page record failed, recall will wait out the deadline"
)

type PageAbsorber interface {
	Absorb(ctx context.Context, page fetchedpage.Page) (pageabsorption.AbsorptionOutcome, error)
}

type DisposedPages interface {
	Record(ctx context.Context, url string) error
}

type Visitor struct {
	fetcher  Fetcher
	recrawl  RecrawlDecision
	absorber PageAbsorber
	disposed DisposedPages
	observer VisitProgress
	clock    clock.Clock
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func New(
	fetcher Fetcher,
	recrawl RecrawlDecision,
	absorber PageAbsorber,
	disposed DisposedPages,
	observer VisitProgress,
	clock clock.Clock,
) *Visitor {
	return &Visitor{
		fetcher:  fetcher,
		recrawl:  recrawl,
		absorber: absorber,
		disposed: disposed,
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
		v.dispose(ctx, url, disposal.NotDue)
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
		v.dispose(ctx, url, disposal.NotModified)
		return VisitOutcome{Conclusion: VisitCompleted}, nil
	case FetchCeased:
		v.recordVisit(ctx, url, outcome.Validators)
		v.observer.RefusalHonored(refusal.Cease)
		v.dispose(ctx, url, disposal.Refused)
		return VisitOutcome{Conclusion: VisitCompleted}, nil
	case FetchDeferred:
		return VisitOutcome{Conclusion: VisitDeferred, DeferFor: outcome.DeferFor}, nil
	case FetchNotAPage:
		v.observer.PageFetched()
		v.dispose(ctx, url, disposal.NotAPage)
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
	absorption, err := v.absorber.Absorb(ctx, outcome.Page)
	if err != nil {
		return VisitOutcome{}, err
	}
	v.recordVisit(ctx, url, outcome.Validators)
	if !absorption.Published {
		v.recordDisposed(ctx, url)
	}
	return VisitOutcome{Conclusion: VisitCompleted, DiscoveredURLs: absorption.DiscoveredURLs}, nil
}

func (v *Visitor) dispose(ctx context.Context, url string, reason disposal.Reason) {
	v.observer.PageDisposed(reason)
	v.recordDisposed(ctx, url)
}

func (v *Visitor) recordDisposed(ctx context.Context, url string) {
	if err := v.disposed.Record(ctx, url); err != nil {
		slog.WarnContext(ctx, msgDisposedPageRecordFailed,
			slog.String("url", url),
			slog.Any("error", err),
		)
	}
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
