// Package orderintake accepts each delivered crawl order and puts its seed URLs
// on the frontier.
package orderintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

type AcceptedOrders interface {
	Keep(ctx context.Context, order acceptedorder.AcceptedOrder) error
}

type PendingVisits interface {
	Publish(ctx context.Context, visit pendingvisit.PendingVisit) error
}

type OrderConsumer struct {
	source                 pullintake.MessageSource
	orders                 AcceptedOrders
	frontier               PendingVisits
	observer               CrawlOrderObserver
	orderIntakeConcurrency int
}

func NewOrderConsumer(
	source pullintake.MessageSource,
	orders AcceptedOrders,
	frontier PendingVisits,
	observer CrawlOrderObserver,
	orderIntakeConcurrency int,
) *OrderConsumer {
	return &OrderConsumer{
		source:                 source,
		orders:                 orders,
		frontier:               frontier,
		observer:               observer,
		orderIntakeConcurrency: orderIntakeConcurrency,
	}
}

func (c *OrderConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.orderIntakeConcurrency, c.acceptOrder)
}

func (c *OrderConsumer) acceptOrder(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	sentOrder, err := yacycrawlcontract.UnmarshalCrawlOrder(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	order, err := acceptedorder.AcceptedOrderFrom(sentOrder)
	if err != nil {
		c.returnOrder(ctx, message, sentOrder.OrderID, err)
		return nil
	}
	if err := c.orders.Keep(ctx, order); err != nil {
		c.returnOrder(ctx, message, order.OrderID(), err)
		return nil
	}
	if err := c.putSeedURLsOnFrontier(ctx, order); err != nil {
		c.returnOrder(ctx, message, order.OrderID(), err)
		return nil
	}
	message.Acknowledge(ctx)
	c.observer.CrawlOrderAccepted(ctx, order.OrderID(), len(order.SeedURLs()))
	return nil
}

func (c *OrderConsumer) putSeedURLsOnFrontier(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
) error {
	for _, seed := range order.SeedURLs() {
		if err := c.frontier.Publish(ctx, pendingvisit.PendingVisit{
			OrderID: order.OrderID(),
			URL:     seed,
			Depth:   0,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *OrderConsumer) returnOrder(
	ctx context.Context,
	message pullintake.PendingMessage,
	orderID string,
	cause error,
) {
	c.observer.CrawlOrderReturned(ctx, orderID, cause)
	message.Return(ctx)
}
