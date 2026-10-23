// Package orderintake takes each delivered crawl order in: it keeps the crawl of a readable
// order and puts its seed URLs on the frontier in one transaction, rejects one it cannot
// read for good, and returns one whose crawl it could not keep yet for redelivery.
package orderintake

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type Crawls interface {
	Keep(tx *vault.Txn, order acceptedorder.AcceptedOrder) (crawlid.CrawlID, error)
}

type Frontier interface {
	Owe(tx *vault.Txn, pageVisits []pendingpagevisit.PendingPageVisit) error
}

type OrderConsumer struct {
	storage                *vault.Vault
	source                 pullintake.MessageSource
	crawls                 Crawls
	frontier               Frontier
	observer               CrawlOrderObserver
	orderIntakeConcurrency int
}

//nolint:revive // an order consumer names the vault and every collaborator it accepts an order with
func NewOrderConsumer(
	storage *vault.Vault,
	source pullintake.MessageSource,
	crawls Crawls,
	frontier Frontier,
	observer CrawlOrderObserver,
	orderIntakeConcurrency int,
) *OrderConsumer {
	return &OrderConsumer{
		storage:                storage,
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
	crawlID, err := c.startCrawl(ctx, order)
	if err != nil {
		c.observer.CrawlOrderReturned(ctx, order.OrderID(), err)
		message.Return(ctx)
		return nil
	}
	message.Acknowledge(ctx)
	c.observer.CrawlOrderAccepted(ctx, order.OrderID(), crawlID, len(order.SeedURLs()))
	return nil
}

func (c *OrderConsumer) startCrawl(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, error) {
	var crawlID crawlid.CrawlID
	err := c.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		if crawlID, err = c.crawls.Keep(tx, order); err != nil {
			return err
		}
		return c.frontier.Owe(tx, seedPageVisitsOf(crawlID, order))
	})
	return crawlID, err
}

func seedPageVisitsOf(
	crawlID crawlid.CrawlID,
	order acceptedorder.AcceptedOrder,
) []pendingpagevisit.PendingPageVisit {
	seedPageVisits := make([]pendingpagevisit.PendingPageVisit, 0, len(order.SeedURLs()))
	for _, seed := range order.SeedURLs() {
		seedPageVisits = append(seedPageVisits, pendingpagevisit.PendingPageVisit{
			CrawlID: crawlID,
			URL:     seed,
			Depth:   0,
		})
	}
	return seedPageVisits
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
