package visitdispatch

import (
	"context"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type CompletedVisit struct {
	Visit   frontier.PendingVisit
	Outcome pagevisit.VisitOutcome
	Err     error
}

type RunningVisitors struct {
	pending   chan frontier.PendingVisit
	completed chan CompletedVisit
	running   sync.WaitGroup
}

func StartVisitors(
	ctx context.Context,
	visitor pagevisit.Visitor,
	concurrency int,
) *RunningVisitors {
	visitors := &RunningVisitors{
		pending:   make(chan frontier.PendingVisit),
		completed: make(chan CompletedVisit, concurrency),
	}
	for range concurrency {
		visitors.running.Go(func() {
			for visit := range visitors.pending {
				outcome, err := visitor.Visit(ctx, visit.URL)
				visitors.completed <- CompletedVisit{Visit: visit, Outcome: outcome, Err: err}
			}
		})
	}

	return visitors
}

func (v *RunningVisitors) Pending() chan<- frontier.PendingVisit {
	return v.pending
}

func (v *RunningVisitors) Completed() <-chan CompletedVisit {
	return v.completed
}

func (v *RunningVisitors) Stop() {
	close(v.pending)
	v.running.Wait()
}
