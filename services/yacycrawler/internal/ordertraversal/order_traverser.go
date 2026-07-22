package ordertraversal

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

type OrderTraverser struct {
	config   Config
	visitor  PageVisitor
	observer crawlcapability.RunProgress
	clock    crawlcapability.Clock
}

func NewOrderTraverser(
	config Config,
	visitor PageVisitor,
	observer crawlcapability.RunProgress,
	clock crawlcapability.Clock,
) *OrderTraverser {
	return &OrderTraverser{
		config:   config,
		visitor:  visitor,
		observer: observer,
		clock:    clock,
	}
}

func (r *OrderTraverser) Traverse(
	ctx context.Context,
	delivery crawlcapability.DeliveredOrder,
) error {
	return r.newOrderTraversal(delivery).run(ctx)
}
