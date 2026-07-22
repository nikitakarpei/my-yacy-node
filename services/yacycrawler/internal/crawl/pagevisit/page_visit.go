package pagevisit

import (
	"context"
	"fmt"
)

type PageAbsorber interface {
	Absorb(ctx context.Context, outcome FetchOutcome) ([]string, error)
}

type PageVisit struct {
	fetch      Fetcher
	recrawl    RecrawlPolicy
	absorption PageAbsorber
	observer   Progress
	clock      Clock
}

func NewPageVisit(
	fetch Fetcher,
	recrawl RecrawlPolicy,
	absorption PageAbsorber,
	observer Progress,
	clock Clock,
) *PageVisit {
	return &PageVisit{
		fetch:      fetch,
		recrawl:    recrawl,
		absorption: absorption,
		observer:   observer,
		clock:      clock,
	}
}

func (v *PageVisit) Visit(ctx context.Context, url string) (VisitOutcome, error) {
	due, err := v.recrawl.Due(ctx, url)
	if err != nil {
		return VisitOutcome{}, fmt.Errorf("recrawl decision: %w", err)
	}
	if !due {
		return VisitOutcome{Classification: NotDue}, nil
	}

	outcome, err := v.fetchPage(ctx, url)
	if err != nil {
		return VisitOutcome{}, err
	}

	switch outcome.Status {
	case FetchCeased:
		v.observer.RefusalHonored(RefusalCease)
		v.observer.PageDisposed(DisposalRefused)
		return VisitOutcome{Classification: Ceased}, nil
	case FetchDeferred:
		return VisitOutcome{Classification: Deferred, DeferFor: outcome.DeferFor}, nil
	case FetchNotAPage:
		v.observer.PageFetched()
		v.observer.PageDisposed(DisposalFetchFailed)
		return VisitOutcome{Classification: NotAPage}, nil
	case FetchTransient:
		return VisitOutcome{Classification: Transient}, nil
	}

	v.observer.PageFetched()
	links, err := v.absorption.Absorb(ctx, outcome)
	if err != nil {
		return VisitOutcome{}, err
	}
	return VisitOutcome{Classification: Succeeded, DiscoveredURLs: links}, nil
}
