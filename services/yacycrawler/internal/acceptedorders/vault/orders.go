// Package vault keeps in a vault every crawl order the crawler accepted, under the
// crawl it runs for that order, for the retention of accepted orders, so that any
// worker can rebuild the profile a URL of that crawl was admitted under. An order
// sent again finds the crawl it began and keeps it.
package vault

import (
	"context"
	"crypto/sha256"
	"errors"
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
	ordersBucket = "accepted_orders"
	crawlsBucket = "crawls_by_order"
)

var errCrawlIDLength = errors.New("stored crawl ID has the wrong length")

var crawlIDKeys = vault.SingleKey(crawlkeyparts.CrawlID).KeyLayout()

var orderIDHashKeys = vault.SingleKey(vault.BytesKeyPartFrom(
	func(hash orderIDHash, buffer []byte) []byte { return append(buffer, hash[:]...) },
	func(stored vault.StoredBytes) (orderIDHash, error) {
		return orderIDHash(stored.AppendTo(nil)), nil
	},
)).KeyLayout()

type Clock interface {
	Now() time.Time
}

type Orders struct {
	storage *vault.Vault
	orders  *retainedrecords.Records[crawlid.CrawlID, yacycrawlcontract.CrawlOrder]
	crawls  *retainedrecords.Records[orderIDHash, crawlid.CrawlID]
	clock   Clock
}

type orderIDHash string

func New(
	ctx context.Context,
	storage *vault.Vault,
	retention *retainedrecords.Retention,
	clock Clock,
	keptFor time.Duration,
) (*Orders, error) {
	orders, err := retention.Register(ctx, ordersBucket, crawlIDKeys, crawlOrderCodec{}, keptFor)
	if err != nil {
		return nil, err
	}
	crawls, err := retention.Register(ctx, crawlsBucket, orderIDHashKeys, crawlIDCodec{}, keptFor)
	if err != nil {
		return nil, err
	}
	return &Orders{storage: storage, orders: orders, crawls: crawls, clock: clock}, nil
}

func (o *Orders) Keep(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, error) {
	now := o.clock.Now()
	var crawlID crawlid.CrawlID
	if err := o.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		crawlID, err = o.crawlOf(tx, order.OrderID(), now)
		if err != nil {
			return err
		}
		return o.keep(tx, crawlID, order, now)
	}); err != nil {
		return crawlid.CrawlID{}, fmt.Errorf("keep order %s: %w", order.OrderID(), err)
	}
	return crawlID, nil
}

func (o *Orders) crawlOf(tx *vault.Txn, orderID string, now time.Time) (crawlid.CrawlID, error) {
	crawlID, found, err := o.crawls.Get(tx, orderIDHashOf(orderID), now)
	if err != nil || found {
		return crawlID, err
	}
	return crawlid.New()
}

func orderIDHashOf(orderID string) orderIDHash {
	digest := sha256.Sum256([]byte(orderID))
	return orderIDHash(digest[:len(crawlid.CrawlID{})])
}

func (o *Orders) keep(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	order acceptedorder.AcceptedOrder,
	now time.Time,
) error {
	if err := o.orders.Put(tx, crawlID, order.CrawlOrder(), now); err != nil {
		return err
	}
	return o.crawls.Put(tx, orderIDHashOf(order.OrderID()), crawlID, now)
}

func (o *Orders) OrderOf(
	ctx context.Context,
	crawlID crawlid.CrawlID,
) (acceptedorder.AcceptedOrder, bool, error) {
	now := o.clock.Now()
	var keptOrder yacycrawlcontract.CrawlOrder
	var found bool
	if err := o.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		keptOrder, found, err = o.orders.Get(tx, crawlID, now)
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

type crawlIDCodec struct{}

func (crawlIDCodec) Encode(crawlID crawlid.CrawlID) ([]byte, error) {
	return crawlID[:], nil
}

func (crawlIDCodec) Decode(stored []byte) (crawlid.CrawlID, error) {
	var crawlID crawlid.CrawlID
	if len(stored) != len(crawlID) {
		return crawlID, fmt.Errorf("%w: %d bytes", errCrawlIDLength, len(stored))
	}
	copy(crawlID[:], stored)
	return crawlID, nil
}
