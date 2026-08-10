package ordertraversal

import (
	"context"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type completedVisit struct {
	visit   frontier.PendingVisit
	outcome pagevisit.VisitOutcome
	err     error
}

type visitors struct {
	pending chan frontier.PendingVisit
	results chan completedVisit
	running sync.WaitGroup
}

func startVisitors(
	ctx context.Context,
	visitor pagevisit.Visitor,
	concurrency int,
) *visitors {
	v := &visitors{
		pending: make(chan frontier.PendingVisit),
		results: make(chan completedVisit, concurrency),
	}
	for range concurrency {
		v.running.Go(func() {
			for visit := range v.pending {
				outcome, err := visitor.Visit(ctx, visit.URL)
				v.results <- completedVisit{visit: visit, outcome: outcome, err: err}
			}
		})
	}
	return v
}

func (v *visitors) stop() {
	close(v.pending)
	v.running.Wait()
}
