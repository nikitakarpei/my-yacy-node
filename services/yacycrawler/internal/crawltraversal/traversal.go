package crawltraversal

import (
	"context"
	"fmt"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlfrontier"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pageadmission"
)

type traversal struct {
	config     Config
	fetch      crawlcapability.PageRetrieval
	recrawl    crawlcapability.RecrawlDecision
	absorption crawlcapability.PageAbsorption
	observer   crawlcapability.RunProgress
	clock      crawlcapability.Clock
	delivery   crawlcapability.DeliveredOrder
	frontier   *crawlfrontier.Frontier
	counted    int
	inflight   int
	fatal      error
}

func (r *Crawler) newTraversal(delivery crawlcapability.DeliveredOrder) *traversal {
	return &traversal{
		config:     r.config,
		fetch:      r.fetch,
		recrawl:    r.recrawl,
		absorption: r.absorption,
		observer:   r.observer,
		clock:      r.clock,
		delivery:   delivery,
	}
}

func (c *traversal) run(ctx context.Context) error {
	seeds := c.canonicalSeeds(ctx, c.delivery.Order.SeedURLs)
	admission, err := pageadmission.New(
		c.delivery.Order.Profile,
		seeds,
		c.config.FrontierCapacity,
	)
	if err != nil {
		return fmt.Errorf("build admission: %w", err)
	}
	c.frontier = crawlfrontier.New(admission)
	for _, seed := range seeds {
		c.frontier.Admit(seed, 0)
	}

	runCtx, cancel := context.WithCancel(ctx)

	dispatch := make(chan crawlfrontier.Entry)
	results := make(chan visitOutcome, c.config.FetchConcurrency)
	var fetchers sync.WaitGroup
	for range c.config.FetchConcurrency {
		fetchers.Add(1)
		go func() {
			defer fetchers.Done()
			for entry := range dispatch {
				results <- c.visit(runCtx, entry)
			}
		}()
	}

	var heartbeat sync.WaitGroup
	if c.config.OwnershipInterval > 0 {
		lease := OwnershipLease{
			extend:   c.delivery.ExtendOwnership,
			interval: c.config.OwnershipInterval,
			clock:    c.clock,
		}
		heartbeat.Add(1)
		go func() {
			defer heartbeat.Done()
			lease.Renew(runCtx)
		}()
	}

	err = c.schedule(runCtx, cancel, dispatch, results)

	close(dispatch)
	fetchers.Wait()
	cancel()
	heartbeat.Wait()
	return err
}
