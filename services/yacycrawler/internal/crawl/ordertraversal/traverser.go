// Package ordertraversal walks one crawl order's URLs within its profile and page budget.
package ordertraversal

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
)

type Traverser struct {
	config   Config
	visitor  PageVisitor
	observer TraversalProgress
	disposed DisposedPages
	clock    clock.Clock
}

func New(
	config Config,
	visitor PageVisitor,
	observer TraversalProgress,
	disposed DisposedPages,
	clock clock.Clock,
) *Traverser {
	return &Traverser{
		config:   config,
		visitor:  visitor,
		observer: observer,
		disposed: disposed,
		clock:    clock,
	}
}

func (t *Traverser) Traverse(
	ctx context.Context,
	order yacycrawlcontract.CrawlOrder,
) error {
	return (&traversal{
		config:   t.config,
		visitor:  t.visitor,
		observer: t.observer,
		disposed: t.disposed,
		clock:    t.clock,
	}).run(ctx, order)
}
