package crawls

import (
	"crypto/sha256"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

type crawlIndexes struct {
	byOrder            *vault.ValueIndex[crawlWithID, crawlOfOrder]
	inState            *vault.ValueIndex[crawlWithID, crawlInState]
	inStateNewestFirst *vault.ValueIndex[crawlWithID, crawlInState]
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
	inStateBucket := crawlsBucket + "_in_state"
	if indexes.inState, err = storage.RegisterValueIndex(
		inStateBucket, crawlIndexesVersion,
		crawlsInStateEntriesOf(crawlsInStateKeyParts), crawlInStateOf,
	); err != nil {
		return crawlIndexes{}, fmt.Errorf("register %s: %w", inStateBucket, err)
	}
	inStateNewestFirstBucket := crawlsBucket + "_in_state_newest_first"
	if indexes.inStateNewestFirst, err = storage.RegisterValueIndex(
		inStateNewestFirstBucket, crawlIndexesVersion,
		crawlsInStateEntriesOf(crawlsInStateNewestFirstKeyParts), crawlInStateOf,
	); err != nil {
		return crawlIndexes{}, fmt.Errorf("register %s: %w", inStateNewestFirstBucket, err)
	}
	return indexes, nil
}

func (indexes crawlIndexes) projectedWith(
	others ...vault.Projection[crawlWithID],
) vault.Projections[crawlid.CrawlID, Crawl] {
	return vault.ProjectedAs(
		func(crawlID crawlid.CrawlID, crawl Crawl) crawlWithID {
			return crawlWithID{crawlID: crawlID, crawl: crawl}
		},
		append(
			[]vault.Projection[crawlWithID]{
				indexes.byOrder, indexes.inState, indexes.inStateNewestFirst,
			},
			others...,
		)...,
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

func (indexes crawlIndexes) crawlIDsOlderThan(
	tx *vault.Txn,
	state yacycrawlcontract.CrawlState,
	border crawlid.CrawlID,
	limit int,
) ([]crawlid.CrawlID, error) {
	keys := crawlsInStateNewestFirstKeyParts.KeysWithFirst(state)
	if border != (crawlid.CrawlID{}) {
		keys = keys.After(crawlsInStateNewestFirstKeyParts.Key(state, border))
	}
	return crawlIDsWithin(tx, indexes.inStateNewestFirst, keys, limit)
}

func (indexes crawlIndexes) crawlIDsNewerThan(
	tx *vault.Txn,
	state yacycrawlcontract.CrawlState,
	border crawlid.CrawlID,
	limit int,
) ([]crawlid.CrawlID, error) {
	keys := crawlsInStateKeyParts.KeysWithFirst(state).
		After(crawlsInStateKeyParts.Key(state, border))
	return crawlIDsWithin(tx, indexes.inState, keys, limit)
}

func crawlIDsWithin(
	tx *vault.Txn,
	index *vault.ValueIndex[crawlWithID, crawlInState],
	keys vault.KeyRange,
	limit int,
) ([]crawlid.CrawlID, error) {
	var crawlIDs []crawlid.CrawlID
	if err := index.Scan(tx, keys, func(entry crawlInState) (bool, error) {
		crawlIDs = append(crawlIDs, entry.crawlID)
		return len(crawlIDs) < limit, nil
	}); err != nil {
		return nil, fmt.Errorf("read crawls: %w", err)
	}
	return crawlIDs, nil
}

type crawlOfOrder struct {
	order   orderIDHash
	crawlID crawlid.CrawlID
}

type orderIDHash string

func crawlOfOrderOf(crawl crawlWithID) crawlOfOrder {
	return crawlOfOrder{order: orderIDHashOf(crawl.crawl.Order.OrderID), crawlID: crawl.crawlID}
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

type crawlInState struct {
	state   yacycrawlcontract.CrawlState
	crawlID crawlid.CrawlID
}

func crawlInStateOf(crawl crawlWithID) crawlInState {
	return crawlInState{state: crawl.crawl.State, crawlID: crawl.crawlID}
}

var (
	crawlsInStateKeyParts            = vault.PairKey(stateKeyPart, vaultkeyparts.CrawlID)
	crawlsInStateNewestFirstKeyParts = vault.PairKey(stateKeyPart, vaultkeyparts.CrawlIDNewestFirst)
)

func crawlsInStateEntriesOf(
	parts vault.PairKeyParts[yacycrawlcontract.CrawlState, crawlid.CrawlID],
) vault.KeyLayout[crawlInState] {
	return parts.KeyLayoutFor(
		func(entry crawlInState) (yacycrawlcontract.CrawlState, crawlid.CrawlID) {
			return entry.state, entry.crawlID
		},
		func(state yacycrawlcontract.CrawlState, crawlID crawlid.CrawlID) crawlInState {
			return crawlInState{state: state, crawlID: crawlID}
		},
	)
}
