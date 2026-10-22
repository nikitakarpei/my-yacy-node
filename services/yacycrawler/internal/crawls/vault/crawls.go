// Package vault keeps in a vault every crawl the crawler runs, with the accepted order
// it runs for, for the retention of crawls, so that any worker can rebuild the profile
// a URL of that crawl was admitted under. An order sent again finds the crawl it began
// and keeps it.
package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	crawlsBucket         = "crawls"
	crawlsByOrderBucket  = "crawls_by_order"
	crawlsByOrderVersion = "1"
)

var crawlIDKeys = vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout()

type Crawls struct {
	storage       *vault.Vault
	crawls        *vault.ExpiringCollection[crawlid.CrawlID, yacycrawlcontract.CrawlOrder]
	crawlsByOrder *vault.ValueIndex[crawlOfOrder, crawlOfOrder]
}

func New(
	ctx context.Context,
	storage *vault.Vault,
	keptFor time.Duration,
) (*Crawls, error) {
	crawlsByOrder, err := storage.RegisterValueIndex(
		crawlsByOrderBucket,
		crawlsByOrderVersion,
		crawlsByOrderEntries,
		func(entry crawlOfOrder) crawlOfOrder { return entry },
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", crawlsByOrderBucket, err)
	}
	crawls, err := storage.RegisterExpiringCollection(
		ctx,
		crawlsBucket,
		crawlIDKeys,
		crawlOrderCodec{},
		keptFor,
		vault.ProjectedAs(crawlOfOrderFrom, crawlsByOrder),
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", crawlsBucket, err)
	}
	return &Crawls{storage: storage, crawls: crawls, crawlsByOrder: crawlsByOrder}, nil
}

func (c *Crawls) Keep(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, error) {
	var crawlID crawlid.CrawlID
	if err := c.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		crawlID, err = c.crawlFor(tx, order.OrderID())
		if err != nil {
			return err
		}
		return c.crawls.Put(tx, crawlID, order.CrawlOrder())
	}); err != nil {
		return crawlid.CrawlID{}, fmt.Errorf("keep order %s: %w", order.OrderID(), err)
	}
	return crawlID, nil
}

func (c *Crawls) crawlFor(tx *vault.Txn, orderID string) (crawlid.CrawlID, error) {
	crawlIDs, err := c.crawlIDsOf(tx, orderIDHashOf(orderID))
	if err != nil {
		return crawlid.CrawlID{}, err
	}
	for _, crawlID := range crawlIDs {
		_, kept, err := c.crawls.Get(tx, crawlID)
		if err != nil || kept {
			return crawlID, err
		}
	}
	return crawlid.New()
}

func (c *Crawls) crawlIDsOf(tx *vault.Txn, order orderIDHash) ([]crawlid.CrawlID, error) {
	var crawlIDs []crawlid.CrawlID
	if err := c.crawlsByOrder.Scan(
		tx,
		crawlsByOrderKeyParts.KeysWithFirst(order),
		func(entry crawlOfOrder) (bool, error) {
			crawlIDs = append(crawlIDs, entry.crawlID)
			return true, nil
		},
	); err != nil {
		return nil, fmt.Errorf("read the crawls of an order: %w", err)
	}
	return crawlIDs, nil
}

func (c *Crawls) OrderOf(
	ctx context.Context,
	crawlID crawlid.CrawlID,
) (acceptedorder.AcceptedOrder, bool, error) {
	var keptOrder yacycrawlcontract.CrawlOrder
	var found bool
	if err := c.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		keptOrder, found, err = c.crawls.Get(tx, crawlID)
		return err
	}); err != nil {
		return acceptedorder.AcceptedOrder{}, false, fmt.Errorf(
			"read the order of crawl %s: %w",
			crawlID,
			err,
		)
	}
	if !found {
		return acceptedorder.AcceptedOrder{}, false, nil
	}
	order, err := acceptedorder.AcceptedOrderFrom(keptOrder)
	if err != nil {
		return acceptedorder.AcceptedOrder{}, false, err
	}
	return order, true, nil
}
