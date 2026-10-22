package vault

import (
	"crypto/sha256"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

type crawlOfOrder struct {
	order   orderIDHash
	crawlID crawlid.CrawlID
}

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

func crawlOfOrderFrom(crawlID crawlid.CrawlID, order yacycrawlcontract.CrawlOrder) crawlOfOrder {
	return crawlOfOrder{order: orderIDHashOf(order.OrderID), crawlID: crawlID}
}

func orderIDHashOf(orderID string) orderIDHash {
	digest := sha256.Sum256([]byte(orderID))
	return orderIDHash(digest[:len(crawlid.CrawlID{})])
}
