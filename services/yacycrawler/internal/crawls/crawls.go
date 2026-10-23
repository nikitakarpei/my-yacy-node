// Package crawls keeps every crawl the crawler runs, with the accepted order it runs
// for, for the retention of crawls, so that any worker can rebuild the profile a URL of
// that crawl was admitted under. An order sent again finds the crawl it began and keeps it.
package crawls

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

type Crawls struct {
	crawls        *vault.ExpiringCollection[crawlid.CrawlID, yacycrawlcontract.CrawlOrder]
	crawlsByOrder *vault.ValueIndex[crawlOfOrder, crawlOfOrder]
}

func Open(ctx context.Context, storage *vault.Vault, keptFor time.Duration) (*Crawls, error) {
	crawls, crawlsByOrder, err := registerCrawls(ctx, storage, keptFor)
	if err != nil {
		return nil, err
	}
	return &Crawls{crawls: crawls, crawlsByOrder: crawlsByOrder}, nil
}

type crawlOfOrder struct {
	order   orderIDHash
	crawlID crawlid.CrawlID
}

func crawlOfOrderFrom(crawlID crawlid.CrawlID, order yacycrawlcontract.CrawlOrder) crawlOfOrder {
	return crawlOfOrder{order: orderIDHashOf(order.OrderID), crawlID: crawlID}
}

func (c *Crawls) Keep(tx *vault.Txn, order acceptedorder.AcceptedOrder) (crawlid.CrawlID, error) {
	crawlID, err := c.crawlFor(tx, order.OrderID())
	if err != nil {
		return crawlid.CrawlID{}, fmt.Errorf("keep order %s: %w", order.OrderID(), err)
	}
	if err := c.crawls.Put(tx, crawlID, order.CrawlOrder()); err != nil {
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
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (acceptedorder.AcceptedOrder, bool, error) {
	keptOrder, found, err := c.crawls.Get(tx, crawlID)
	if err != nil {
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
