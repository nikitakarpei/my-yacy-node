package ordertraversal

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/contextcancellation"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

func (c *orderTraversal) schedule(
	ctx context.Context,
	cancel context.CancelFunc,
	dispatch chan entry,
	results chan pageVisitResult,
) error {
	budget := c.config.RunPageBudget
	for {
		if c.fatal != nil {
			return c.drainInflight(results)
		}
		if c.counted >= budget && c.inflight == 0 {
			c.disposePendingOverBudget()
			return nil
		}
		if c.frontier.Drained() && c.inflight == 0 {
			return nil
		}

		var dispatchable chan entry
		var next entry
		if c.dispatchable(budget) {
			next, _ = c.frontier.Peek()
			dispatchable = dispatch
		} else if c.inflight == 0 {
			if err := c.awaitEarliestDue(ctx); err != nil {
				c.fatal = err
				cancel()
			}
			continue
		}

		select {
		case dispatchable <- next:
			c.frontier.Next()
			c.inflight++
		case r := <-results:
			c.recordVisit(ctx, r, cancel)
		case <-ctx.Done():
			c.fatal = contextcancellation.Err(ctx)
			cancel()
		}
	}
}

func (c *orderTraversal) dispatchable(budget int) bool {
	if c.fatal != nil {
		return false
	}
	if c.counted+c.inflight >= budget {
		return false
	}
	return c.frontier.HasReady()
}

func (c *orderTraversal) awaitEarliestDue(ctx context.Context) error {
	due, ok := c.frontier.EarliestDue()
	if !ok {
		return nil
	}
	wait := due.Sub(c.clock.Now())
	if wait > 0 {
		if err := c.clock.Sleep(ctx, wait); err != nil {
			return fmt.Errorf("await earliest: %w", err)
		}
	}
	c.frontier.PromoteDue(c.clock.Now())
	return nil
}

func (c *orderTraversal) drainInflight(results chan pageVisitResult) error {
	for c.inflight > 0 {
		<-results
		c.inflight--
	}
	return c.fatal
}

func (c *orderTraversal) disposePendingOverBudget() {
	c.observer.BudgetExhausted()
	for range c.frontier.DrainPending() {
		c.observer.PageDisposed(crawlcapability.DisposalBudgetTruncated)
	}
}
