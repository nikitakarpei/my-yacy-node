package pagevisit

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type PageVisit struct {
	fetch      crawlcapability.PageRetrieval
	recrawl    crawlcapability.RecrawlDecision
	absorption crawlcapability.PageAbsorption
	observer   crawlcapability.RunProgress
	clock      crawlcapability.Clock
}

func NewPageVisit(
	fetch crawlcapability.PageRetrieval,
	recrawl crawlcapability.RecrawlDecision,
	absorption crawlcapability.PageAbsorption,
	observer crawlcapability.RunProgress,
	clock crawlcapability.Clock,
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
	case crawlcapability.FetchCeased:
		v.observer.RefusalHonored(crawlcapability.RefusalCease)
		v.observer.PageDisposed(crawlcapability.DisposalRefused)
		return VisitOutcome{Classification: Ceased}, nil
	case crawlcapability.FetchDeferred:
		return VisitOutcome{Classification: Deferred, DeferFor: outcome.DeferFor}, nil
	case crawlcapability.FetchNotAPage:
		v.observer.PageFetched()
		v.observer.PageDisposed(crawlcapability.DisposalFetchFailed)
		return VisitOutcome{Classification: NotAPage}, nil
	case crawlcapability.FetchTransient:
		return VisitOutcome{Classification: Transient}, nil
	}

	v.observer.PageFetched()
	links, err := v.absorption.Absorb(ctx, outcome)
	if err != nil {
		return VisitOutcome{}, err
	}
	return VisitOutcome{Classification: Succeeded, DiscoveredURLs: links}, nil
}
