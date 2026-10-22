package crawls

import (
	"crypto/sha256"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

type crawlIndexes struct {
	byOrder *vault.ValueIndex[crawlForm, crawlOfOrder]
}

func registerCrawlIndexes(storage *vault.Vault, crawlsBucket vault.Name) (crawlIndexes, error) {
	var indexes crawlIndexes
	var err error
	byOrderBucket := crawlsBucket + "_by_order"
	if indexes.byOrder, err = storage.RegisterValueIndex(
		byOrderBucket, crawlIndexesVersion, crawlsByOrderEntries, crawlOfOrderOf,
	); err != nil {
		return crawlIndexes{}, fmt.Errorf("register %s: %w", byOrderBucket, err)
	}
	return indexes, nil
}

func (indexes crawlIndexes) projections() vault.Projections[crawlid.CrawlID, Crawl] {
	return vault.ProjectedAs(
		func(crawlID crawlid.CrawlID, crawl Crawl) crawlForm {
			return crawlForm{crawlID: crawlID, crawl: crawl}
		},
		indexes.byOrder,
	)
}

func (indexes crawlIndexes) crawlIDsOfOrder(
	tx *vault.Txn,
	orderID string,
) ([]crawlid.CrawlID, error) {
	var crawlIDs []crawlid.CrawlID
	if err := indexes.byOrder.Scan(
		tx,
		crawlsByOrderKeyParts.KeysWithFirst(orderIDHashOf(orderID)),
		func(entry crawlOfOrder) (bool, error) {
			crawlIDs = append(crawlIDs, entry.crawlID)
			return true, nil
		},
	); err != nil {
		return nil, fmt.Errorf("read the crawls of order %s: %w", orderID, err)
	}
	return crawlIDs, nil
}

type crawlOfOrder struct {
	order   orderIDHash
	crawlID crawlid.CrawlID
}

type orderIDHash string

func crawlOfOrderOf(form crawlForm) crawlOfOrder {
	return crawlOfOrder{order: orderIDHashOf(form.crawl.Order.OrderID), crawlID: form.crawlID}
}

func orderIDHashOf(orderID string) orderIDHash {
	digest := sha256.Sum256([]byte(orderID))
	return orderIDHash(digest[:len(crawlid.CrawlID{})])
}

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
