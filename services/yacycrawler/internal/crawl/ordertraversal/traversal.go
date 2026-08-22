package ordertraversal

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/contextcancellation"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/frontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/profileadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitdispatch"
)

const (
	msgDeferralsExhausted = "url dropped after exhausting deferrals"
	msgFetchAbandoned     = "fetch abandoned after retries"
)

type traversal struct {
	config         Config
	visitorSource  pagevisit.VisitorSource
	observer       TraversalProgress
	disposer       *disposal.Disposer
	clock          clock.Clock
	cancel         context.CancelFunc
	frontier       *frontier.Frontier
	visitors       *visitdispatch.RunningVisitors
	fetchedPages   int
	inflightVisits int
	abortErr       error
}

func (t *traversal) run(ctx context.Context, order yacycrawlcontract.CrawlOrder) error {
	admission, err := profileadmission.New(order.Profile, order.SeedURLs, t.config.MaxAdmittedURLs)
	if err != nil {
		return fmt.Errorf("build admission: %w", err)
	}
	t.frontier = frontier.New(admission, order.SeedURLs, t.config.Frontier)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	t.cancel = cancel

	visitor := t.visitorSource.VisitorFor(indexingRefusalOf(order.Profile))
	t.visitors = visitdispatch.StartVisitors(runCtx, visitor, t.config.VisitConcurrency)
	err = t.schedule(runCtx)
	t.visitors.Stop()
	return err
}

func indexingRefusalOf(
	profile yacycrawlcontract.CrawlProfile,
) pagevisit.IndexingRefusal {
	if profile.IgnoresIndexingRefusal {
		return pagevisit.Ignored
	}
	return pagevisit.Honored
}

func (t *traversal) schedule(ctx context.Context) error {
	budget := t.config.RunPageBudget
	for {
		if t.abortErr != nil {
			return t.drainInflight()
		}
		if t.fetchedPages >= budget && t.inflightVisits == 0 {
			t.disposePendingOverBudget(ctx)
			return nil
		}
		if t.frontier.Empty() && t.inflightVisits == 0 {
			return nil
		}

		var toVisit chan<- frontier.PendingVisit
		next, ready := t.readyVisit(budget)
		switch {
		case ready:
			toVisit = t.visitors.Pending()
		case t.inflightVisits == 0:
			t.awaitEarliestDue(ctx)
			continue
		}

		select {
		case toVisit <- next:
			t.frontier.Drop()
			t.inflightVisits++
		case result := <-t.visitors.Completed():
			t.recordVisit(ctx, result)
		case <-ctx.Done():
			t.abort(contextcancellation.Err(ctx))
		}
	}
}

func (t *traversal) readyVisit(budget int) (frontier.PendingVisit, bool) {
	if t.fetchedPages+t.inflightVisits >= budget {
		return frontier.PendingVisit{}, false
	}
	return t.frontier.Peek()
}

func (t *traversal) awaitEarliestDue(ctx context.Context) {
	due, ok := t.frontier.EarliestDue()
	if !ok {
		return
	}
	if wait := due.Sub(t.clock.Now()); wait > 0 {
		if err := t.clock.Sleep(ctx, wait); err != nil {
			t.abort(fmt.Errorf("await earliest: %w", err))
			return
		}
	}
	t.frontier.PromoteDue(t.clock.Now())
}

func (t *traversal) drainInflight() error {
	for t.inflightVisits > 0 {
		<-t.visitors.Completed()
		t.inflightVisits--
	}
	return t.abortErr
}

func (t *traversal) disposePendingOverBudget(ctx context.Context) {
	t.observer.BudgetExhausted()
	for _, pending := range t.frontier.DrainPending() {
		t.disposer.Dispose(ctx, pending.URL, disposal.BudgetTruncated)
	}
}

func (t *traversal) recordVisit(ctx context.Context, result visitdispatch.CompletedVisit) {
	t.inflightVisits--
	if result.Err != nil {
		t.abort(result.Err)
		return
	}
	if result.Outcome.Fetched {
		t.fetchedPages++
	}
	switch result.Outcome.Conclusion {
	case pagevisit.VisitDeferred:
		t.recordDeferred(ctx, result.Visit, result.Outcome.DeferFor)
	case pagevisit.VisitRetryable:
		t.recordRetryable(ctx, result.Visit)
	case pagevisit.VisitCompleted:
		t.recordCompleted(ctx, result.Visit, result.Outcome)
	}
}

func (t *traversal) recordDeferred(
	ctx context.Context,
	visit frontier.PendingVisit,
	deferFor time.Duration,
) {
	if !t.frontier.Defer(visit, t.clock.Now(), deferFor) {
		slog.WarnContext(ctx, msgDeferralsExhausted, slog.String("url", visit.URL.String()))
		t.disposer.Dispose(ctx, visit.URL, disposal.DeferralsExhausted)
		return
	}
	t.observer.RefusalHonored(refusal.Defer)
}

func (t *traversal) recordRetryable(ctx context.Context, visit frontier.PendingVisit) {
	if !t.frontier.Retry(visit, t.clock.Now()) {
		slog.WarnContext(ctx, msgFetchAbandoned, slog.String("url", visit.URL.String()))
		t.disposer.Dispose(ctx, visit.URL, disposal.FetchAbandoned)
	}
}

func (t *traversal) recordCompleted(
	ctx context.Context,
	visit frontier.PendingVisit,
	outcome pagevisit.VisitOutcome,
) {
	if outcome.Disposal != disposal.NotDisposed {
		t.disposer.Dispose(ctx, visit.URL, outcome.Disposal)
	}
	for _, url := range outcome.DiscoveredURLs {
		t.frontier.Admit(url, visit.Depth+1)
	}
}

func (t *traversal) abort(err error) {
	if t.abortErr == nil {
		t.abortErr = err
	}
	t.cancel()
}
