package crawls

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	crawlsBucket         = "crawls"
	crawlsByOrderBucket  = "crawls_by_order"
	crawlsByOrderVersion = "1"
)

func registerCrawls(
	ctx context.Context,
	storage *vault.Vault,
	keptFor time.Duration,
) (
	*vault.ExpiringCollection[crawlid.CrawlID, yacycrawlcontract.CrawlOrder],
	*vault.ValueIndex[crawlOfOrder, crawlOfOrder],
	error,
) {
	crawlsByOrder, err := storage.RegisterValueIndex(
		crawlsByOrderBucket,
		crawlsByOrderVersion,
		crawlsByOrderEntries,
		func(entry crawlOfOrder) crawlOfOrder { return entry },
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register %s: %w", crawlsByOrderBucket, err)
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
		return nil, nil, fmt.Errorf("register %s: %w", crawlsBucket, err)
	}
	return crawls, crawlsByOrder, nil
}

var crawlIDKeys = vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout()

type orderIDHash string

var crawlsByOrderKeyParts = vault.PairKey(vault.BytesKeyPartFrom(
	func(hash orderIDHash, buffer []byte) []byte { return append(buffer, hash[:]...) },
	func(stored vault.StoredBytes) (orderIDHash, error) {
		return orderIDHash(stored.AppendTo(nil)), nil
	},
), vaultkeyparts.CrawlID)

var crawlsByOrderEntries = crawlsByOrderKeyParts.KeyLayoutFor(
	func(entry crawlOfOrder) (orderIDHash, crawlid.CrawlID) { return entry.order, entry.crawlID },
	func(order orderIDHash, crawlID crawlid.CrawlID) crawlOfOrder {
		return crawlOfOrder{order: order, crawlID: crawlID}
	},
)

func orderIDHashOf(orderID string) orderIDHash {
	digest := sha256.Sum256([]byte(orderID))
	return orderIDHash(digest[:len(crawlid.CrawlID{})])
}

type crawlOrderCodec struct{}

func (crawlOrderCodec) Encode(order yacycrawlcontract.CrawlOrder) ([]byte, error) {
	return yacycrawlcontract.MarshalCrawlOrder(order)
}

func (crawlOrderCodec) Decode(stored []byte) (yacycrawlcontract.CrawlOrder, error) {
	return yacycrawlcontract.UnmarshalCrawlOrder(stored)
}
