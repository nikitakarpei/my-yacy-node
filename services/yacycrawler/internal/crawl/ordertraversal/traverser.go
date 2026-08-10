// Package ordertraversal walks one crawl order's URLs within its profile and page budget.
package ordertraversal

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
)

type Traverser struct {
	config        Config
	visitorSource pagevisit.VisitorSource
	observer      TraversalProgress
	disposer      *disposal.Disposer
	clock         clock.Clock
}

func New(
	config Config,
	visitorSource pagevisit.VisitorSource,
	observer TraversalProgress,
	disposer *disposal.Disposer,
	clock clock.Clock,
) *Traverser {
	return &Traverser{
		config:        config,
		visitorSource: visitorSource,
		observer:      observer,
		disposer:      disposer,
		clock:         clock,
	}
}

func (t *Traverser) Traverse(
	ctx context.Context,
	order yacycrawlcontract.CrawlOrder,
) error {
	return (&traversal{
		config:        t.config,
		visitorSource: t.visitorSource,
		observer:      t.observer,
		disposer:      t.disposer,
		clock:         t.clock,
	}).run(ctx, order)
}
