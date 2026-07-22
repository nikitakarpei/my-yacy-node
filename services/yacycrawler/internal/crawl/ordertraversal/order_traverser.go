package ordertraversal

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/ordersettlement"
)

type OrderTraverser struct {
	config   Config
	visitor  PageVisitor
	observer Progress
	clock    Clock
}

func NewOrderTraverser(
	config Config,
	visitor PageVisitor,
	observer Progress,
	clock Clock,
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
	delivery ordersettlement.DeliveredOrder,
) error {
	return r.newOrderTraversal(delivery).run(ctx)
}
