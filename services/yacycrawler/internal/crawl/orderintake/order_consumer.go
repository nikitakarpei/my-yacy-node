// Package orderintake accepts each delivered crawl order and puts its seed URLs
// on the frontier.
package orderintake

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

const (
	msgOrderAccepted    = "crawl order accepted"
	msgOrderRedelivered = "crawl order returned for redelivery"
)

type AcceptedOrders interface {
	Accept(ctx context.Context, order yacycrawlcontract.CrawlOrder) error
}

type PendingVisits interface {
	Publish(ctx context.Context, visit pendingvisit.PendingVisit) error
}

type OrderProgress interface {
	OrderReceived()
	OrderRedelivered()
	OrderCompleted()
}

type Config struct {
	Source                 pullintake.MessageSource
	Orders                 AcceptedOrders
	Visits                 PendingVisits
	Observer               OrderProgress
	OrderIntakeConcurrency int
}

type OrderConsumer struct {
	source                 pullintake.MessageSource
	orders                 AcceptedOrders
	visits                 PendingVisits
	observer               OrderProgress
	orderIntakeConcurrency int
}

func NewOrderConsumer(config Config) *OrderConsumer {
	return &OrderConsumer{
		source:                 config.Source,
		orders:                 config.Orders,
		visits:                 config.Visits,
		observer:               config.Observer,
		orderIntakeConcurrency: config.OrderIntakeConcurrency,
	}
}

func (c *OrderConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.orderIntakeConcurrency, c.processOne)
}

func (c *OrderConsumer) processOne(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	order, err := yacycrawlcontract.UnmarshalCrawlOrder(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	c.observer.OrderReceived()
	if err := c.orders.Accept(ctx, order); err != nil {
		c.redeliver(ctx, message, order.OrderID, err)
		return nil
	}
	if err := c.seed(ctx, order); err != nil {
		c.redeliver(ctx, message, order.OrderID, err)
		return nil
	}
	message.Acknowledge(ctx)
	c.observer.OrderCompleted()
	slog.DebugContext(ctx, msgOrderAccepted,
		slog.String("order", order.OrderID),
		slog.Int("seeds", len(order.SeedURLs)),
	)
	return nil
}

func (c *OrderConsumer) seed(ctx context.Context, order yacycrawlcontract.CrawlOrder) error {
	for _, seed := range order.SeedURLs {
		if err := c.visits.Publish(ctx, pendingvisit.PendingVisit{
			OrderID: order.OrderID,
			URL:     seed,
			Depth:   0,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *OrderConsumer) redeliver(
	ctx context.Context,
	message pullintake.PendingMessage,
	orderID string,
	cause error,
) {
	c.observer.OrderRedelivered()
	slog.WarnContext(ctx, msgOrderRedelivered,
		slog.String("order", orderID),
		slog.Any("error", cause),
	)
	message.Return(ctx)
}
