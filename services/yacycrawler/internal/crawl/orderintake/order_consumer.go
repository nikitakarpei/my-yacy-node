// Package orderintake accepts each delivered crawl order and puts its seed URLs
// on the frontier.
package orderintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type Crawls interface {
	Keep(ctx context.Context, order acceptedorder.AcceptedOrder) (crawlid.CrawlID, error)
}

type PendingPageVisits interface {
	Publish(ctx context.Context, pageVisits []pendingpagevisit.PendingPageVisit) error
}

type OrderConsumer struct {
	source                 pullintake.MessageSource
	crawls                 Crawls
	frontier               PendingPageVisits
	observer               CrawlOrderObserver
	orderIntakeConcurrency int
}

func NewOrderConsumer(
	source pullintake.MessageSource,
	crawls Crawls,
	frontier PendingPageVisits,
	observer CrawlOrderObserver,
	orderIntakeConcurrency int,
) *OrderConsumer {
	return &OrderConsumer{
		source:                 source,
		crawls:                 crawls,
		frontier:               frontier,
		observer:               observer,
		orderIntakeConcurrency: orderIntakeConcurrency,
	}
}

func (c *OrderConsumer) Run(ctx context.Context) error {
	return pullintake.Run(
		ctx,
		pullintake.PendingMessagesOf(c.source),
		c.orderIntakeConcurrency,
		c.acceptOrder,
	)
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
	crawlID, err := c.crawls.Keep(ctx, order)
	if err != nil {
		c.returnOrder(ctx, message, order.OrderID(), err)
		return nil
	}
	if err := c.putSeedURLsOnFrontier(ctx, crawlID, order); err != nil {
		c.returnOrder(ctx, message, order.OrderID(), err)
		return nil
	}
	message.Acknowledge(ctx)
	c.observer.CrawlOrderAccepted(ctx, order.OrderID(), crawlID, len(order.SeedURLs()))
	return nil
}

func (c *OrderConsumer) putSeedURLsOnFrontier(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	order acceptedorder.AcceptedOrder,
) error {
	seedPageVisits := make([]pendingpagevisit.PendingPageVisit, 0, len(order.SeedURLs()))
	for _, seed := range order.SeedURLs() {
		seedPageVisits = append(seedPageVisits, pendingpagevisit.PendingPageVisit{
			CrawlID: crawlID,
			URL:     seed,
			Depth:   0,
		})
	}
	return c.frontier.Publish(ctx, seedPageVisits)
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
