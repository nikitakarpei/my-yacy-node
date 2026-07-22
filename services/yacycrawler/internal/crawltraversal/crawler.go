package crawltraversal

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type Crawler struct {
	config     Config
	fetch      crawlcapability.PageRetrieval
	recrawl    crawlcapability.RecrawlDecision
	absorption crawlcapability.PageAbsorption
	observer   crawlcapability.RunProgress
	clock      crawlcapability.Clock
}

//nolint:revive // argument-limit: the crawler's collaborators are all distinct ports.
func NewCrawler(
	config Config,
	fetch crawlcapability.PageRetrieval,
	recrawl crawlcapability.RecrawlDecision,
	absorption crawlcapability.PageAbsorption,
	observer crawlcapability.RunProgress,
	clock crawlcapability.Clock,
) *Crawler {
	return &Crawler{
		config:     config,
		fetch:      fetch,
		recrawl:    recrawl,
		absorption: absorption,
		observer:   observer,
		clock:      clock,
	}
}

func (r *Crawler) Traverse(ctx context.Context, delivery crawlcapability.DeliveredOrder) error {
	return r.newTraversal(delivery).run(ctx)
}
