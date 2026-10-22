// Package orderintake takes each delivered crawl order in: it starts the crawl of a
// readable order with its seeds pending, rejects one it cannot read for good, and returns one
// whose crawl it could not start yet for redelivery.
package orderintake

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type Crawls interface {
	Start(
		tx *vault.Txn,
		order acceptedorder.AcceptedOrder,
	) (crawlID crawlid.CrawlID, started bool, err error)
}

type PendingPageVisits interface {
	Add(tx *vault.Txn, pageVisits []pendingpagevisit.PendingPageVisit) error
}

type OrderConsumer struct {
	source                 pullintake.MessageSource
	storage                *vault.Vault
	crawls                 Crawls
	pending                PendingPageVisits
	observer               CrawlOrderObserver
	orderIntakeConcurrency int
}

//nolint:revive // a consumer names its source, the vault, the stores it starts a crawl in and more
func NewOrderConsumer(
	source pullintake.MessageSource,
	storage *vault.Vault,
	crawls Crawls,
	pending PendingPageVisits,
	observer CrawlOrderObserver,
	orderIntakeConcurrency int,
) *OrderConsumer {
	return &OrderConsumer{
		source:                 source,
		storage:                storage,
		crawls:                 crawls,
		pending:                pending,
		observer:               observer,
		orderIntakeConcurrency: orderIntakeConcurrency,
	}
}

func (c *OrderConsumer) Run(ctx context.Context) error {
	return pullintake.Run(
		ctx,
		pullintake.PendingMessagesOf(c.source),
		c.orderIntakeConcurrency,
		c.takeInOrder,
	)
}

func (c *OrderConsumer) takeInOrder(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	sentOrder, err := yacycrawlcontract.UnmarshalCrawlOrder(message.Body())
	if err != nil {
		c.rejectOrder(ctx, message, "", err)
		return nil
	}
	order, err := acceptedorder.AcceptedOrderFrom(sentOrder)
	if err != nil {
		c.rejectOrder(ctx, message, sentOrder.OrderID, err)
		return nil
	}
	crawlID, started, err := c.startCrawl(ctx, order)
	if err != nil {
		c.observer.CrawlOrderReturned(ctx, order.OrderID(), err)
		message.Return(ctx)
		return nil
	}
	message.Acknowledge(ctx)
	c.observeAcceptance(ctx, order, crawlID, started)
	return nil
}

func (c *OrderConsumer) startCrawl(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, bool, error) {
	var crawlID crawlid.CrawlID
	var started bool
	if err := c.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		crawlID, started, err = c.crawls.Start(tx, order)
		if err != nil || !started {
			return err
		}
		return c.pending.Add(tx, pendingpagevisit.SeedPageVisitsOf(crawlID, order.SeedURLs()))
	}); err != nil {
		return crawlid.CrawlID{}, false, fmt.Errorf(
			"start the crawl of order %s: %w",
			order.OrderID(),
			err,
		)
	}
	return crawlID, started, nil
}

func (c *OrderConsumer) rejectOrder(
	ctx context.Context,
	message pullintake.PendingMessage,
	orderID string,
	cause error,
) {
	message.Acknowledge(ctx)
	c.observer.CrawlOrderRejected(ctx, message.Identity(), orderID, cause)
}

func (c *OrderConsumer) observeAcceptance(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
	crawlID crawlid.CrawlID,
	started bool,
) {
	if !started {
		c.observer.CrawlOrderIgnored(ctx, order.OrderID())
		return
	}
	c.observer.CrawlOrderAccepted(ctx, order.OrderID(), crawlID, len(order.SeedURLs()))
}
