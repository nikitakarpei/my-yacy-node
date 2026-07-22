package ordertraversal

import (
	"context"
	"fmt"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordersettlement"
)

type orderTraversal struct {
	config   Config
	visitor  PageVisitor
	observer Progress
	clock    Clock
	delivery ordersettlement.DeliveredOrder
	frontier *frontier
	counted  int
	inflight int
	fatal    error
}

func (r *OrderTraverser) newOrderTraversal(
	delivery ordersettlement.DeliveredOrder,
) *orderTraversal {
	return &orderTraversal{
		config:   r.config,
		visitor:  r.visitor,
		observer: r.observer,
		clock:    r.clock,
		delivery: delivery,
	}
}

func (c *orderTraversal) run(ctx context.Context) error {
	seeds := c.canonicalSeeds(ctx, c.delivery.Order.SeedURLs)
	admission, err := newProfileAdmission(
		c.delivery.Order.Profile,
		seeds,
		c.config.FrontierCapacity,
	)
	if err != nil {
		return fmt.Errorf("build admission: %w", err)
	}
	c.frontier = newFrontier(admission)
	for _, seed := range seeds {
		c.frontier.Admit(seed, 0)
	}

	runCtx, cancel := context.WithCancel(ctx)

	dispatch := make(chan entry)
	results := make(chan pageVisitResult, c.config.FetchConcurrency)
	var fetchers sync.WaitGroup
	for range c.config.FetchConcurrency {
		fetchers.Add(1)
		go func() {
			defer fetchers.Done()
			for next := range dispatch {
				outcome, visitErr := c.visitor.Visit(runCtx, next.URL)
				results <- pageVisitResult{entry: next, outcome: outcome, err: visitErr}
			}
		}()
	}

	err = c.schedule(runCtx, cancel, dispatch, results)

	close(dispatch)
	fetchers.Wait()
	cancel()
	return err
}
