// Package vault keeps in a vault every crawl the crawler runs, with the accepted order
// it runs for, for the retention of crawls, so that any worker can rebuild the profile
// a URL of that crawl was admitted under. An order sent again finds the crawl it began
// and keeps it.
package vault

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlkeyparts"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/retainedrecords"
)

const (
	crawlsBucket         = "crawls"
	crawlsByOrderBucket  = "crawls_by_order"
	crawlsByOrderVersion = "1"
)

var crawlIDKeys = vault.SingleKey(crawlkeyparts.CrawlID).KeyLayout()

var crawlsByOrderKeyParts = vault.PairKey(vault.BytesKeyPartFrom(
	func(hash orderIDHash, buffer []byte) []byte { return append(buffer, hash[:]...) },
	func(stored vault.StoredBytes) (orderIDHash, error) {
		return orderIDHash(stored.AppendTo(nil)), nil
	},
), crawlkeyparts.CrawlID)

var crawlsByOrderEntries = crawlsByOrderKeyParts.KeyLayoutFor(
	func(entry crawlOfOrder) (orderIDHash, crawlid.CrawlID) { return entry.order, entry.crawlID },
	func(order orderIDHash, crawlID crawlid.CrawlID) crawlOfOrder {
		return crawlOfOrder{order: order, crawlID: crawlID}
	},
)

type Clock interface {
	Now() time.Time
}

type Crawls struct {
	storage       *vault.Vault
	crawls        *retainedrecords.Records[crawlid.CrawlID, yacycrawlcontract.CrawlOrder]
	crawlsByOrder *vault.ValueIndex[retainedOrder, crawlOfOrder]
	clock         Clock
}

type retainedOrder = retainedrecords.Record[crawlid.CrawlID, yacycrawlcontract.CrawlOrder]

type orderIDHash string

type crawlOfOrder struct {
	order   orderIDHash
	crawlID crawlid.CrawlID
}

func New(
	ctx context.Context,
	storage *vault.Vault,
	retention *retainedrecords.Retention,
	clock Clock,
	keptFor time.Duration,
) (*Crawls, error) {
	crawlsByOrder, err := storage.RegisterValueIndex(
		crawlsByOrderBucket,
		crawlsByOrderVersion,
		crawlsByOrderEntries,
		crawlOfOrderFrom,
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", crawlsByOrderBucket, err)
	}
	crawls, err := retention.Register(
		ctx, crawlsBucket, crawlIDKeys, crawlOrderCodec{}, keptFor, crawlsByOrder,
	)
	if err != nil {
		return nil, err
	}
	return &Crawls{
		storage:       storage,
		crawls:        crawls,
		crawlsByOrder: crawlsByOrder,
		clock:         clock,
	}, nil
}

func crawlOfOrderFrom(order retainedOrder) crawlOfOrder {
	return crawlOfOrder{order: orderIDHashOf(order.Value.OrderID), crawlID: order.Key}
}

func orderIDHashOf(orderID string) orderIDHash {
	digest := sha256.Sum256([]byte(orderID))
	return orderIDHash(digest[:len(crawlid.CrawlID{})])
}

func (c *Crawls) Keep(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, error) {
	now := c.clock.Now()
	var crawlID crawlid.CrawlID
	if err := c.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		crawlID, err = c.crawlFor(tx, order.OrderID(), now)
		if err != nil {
			return err
		}
		return c.crawls.Put(tx, crawlID, order.CrawlOrder(), now)
	}); err != nil {
		return crawlid.CrawlID{}, fmt.Errorf("keep order %s: %w", order.OrderID(), err)
	}
	return crawlID, nil
}

func (c *Crawls) crawlFor(tx *vault.Txn, orderID string, now time.Time) (crawlid.CrawlID, error) {
	crawlIDs, err := c.crawlIDsOf(tx, orderIDHashOf(orderID))
	if err != nil {
		return crawlid.CrawlID{}, err
	}
	for _, crawlID := range crawlIDs {
		_, kept, err := c.crawls.Get(tx, crawlID, now)
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
	now := c.clock.Now()
	var keptOrder yacycrawlcontract.CrawlOrder
	var found bool
	if err := c.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		keptOrder, found, err = c.crawls.Get(tx, crawlID, now)
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

type crawlOrderCodec struct{}

func (crawlOrderCodec) Encode(order yacycrawlcontract.CrawlOrder) ([]byte, error) {
	return yacycrawlcontract.MarshalCrawlOrder(order)
}

func (crawlOrderCodec) Decode(stored []byte) (yacycrawlcontract.CrawlOrder, error) {
	return yacycrawlcontract.UnmarshalCrawlOrder(stored)
}
