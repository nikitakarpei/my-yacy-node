package frontier

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/urlhash"
)

const (
	pendingPageVisitsBucket        = "pending_page_visits"
	pendingPageVisitsByDueBucket   = "pending_page_visits_by_due"
	pendingPageVisitsOfCrawlBucket = "pending_page_visits_of_crawl"
	pendingPageVisitsOfHostBucket  = "pending_page_visits_of_host"
	endedPageVisitsBucket          = "ended_page_visits"
)

var errPendingPageVisitRecordMalformed = errors.New(
	"pending page visit record has a malformed field",
)

type pageVisitLedger struct {
	pending        *vault.Collection[pageVisitKey, pendingPageVisitRecord]
	pendingByDue   *vault.ValueIndex[pendingPageVisitWithKey, dueEntry]
	pendingOfCrawl *vault.CountIndex[pendingPageVisitWithKey, crawlid.CrawlID]
	pendingOfHost  *vault.CountIndex[pendingPageVisitWithKey, crawlHost]
	ended          *vault.Collection[pageVisitKey, endedMark]
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	urlHash urlhash.URLHash
}

type pendingPageVisitRecord struct {
	due   int64
	depth int
	url   canonicalurl.CanonicalURL
}

type endedMark struct{}

func registerPageVisitLedger(ctx context.Context, storage *vault.Vault) (pageVisitLedger, error) {
	var ledger pageVisitLedger
	var err error
	if ledger.pendingByDue, err = storage.RegisterValueIndex(
		pendingPageVisitsByDueBucket,
		frontierIndexesVersion,
		dueEntries,
		func(pending pendingPageVisitWithKey) dueEntry {
			return dueEntry{key: pending.key, due: pending.visit.due}
		},
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", pendingPageVisitsByDueBucket, err)
	}
	if ledger.pendingOfCrawl, err = storage.RegisterCountIndex(
		pendingPageVisitsOfCrawlBucket, frontierIndexesVersion,
		vault.SingleKey(crawlid.KeyPart).KeyLayout(),
		func(pending pendingPageVisitWithKey) crawlid.CrawlID { return pending.key.crawlID },
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", pendingPageVisitsOfCrawlBucket, err)
	}
	if ledger.pendingOfHost, err = storage.RegisterCountIndex(
		pendingPageVisitsOfHostBucket, frontierIndexesVersion, crawlHosts,
		func(pending pendingPageVisitWithKey) crawlHost {
			return crawlHost{crawlID: pending.key.crawlID, host: pending.visit.url.Hostname()}
		},
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", pendingPageVisitsOfHostBucket, err)
	}
	if ledger.pending, err = storage.RegisterProjectedCollection(
		ctx, pendingPageVisitsBucket, pageVisitKeys, pendingPageVisitRecordCodec{},
		vault.ProjectedAs(
			func(key pageVisitKey, visit pendingPageVisitRecord) pendingPageVisitWithKey {
				return pendingPageVisitWithKey{key: key, visit: visit}
			},
			ledger.pendingByDue, ledger.pendingOfCrawl, ledger.pendingOfHost,
		),
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", pendingPageVisitsBucket, err)
	}
	if ledger.ended, err = storage.RegisterCollection(
		endedPageVisitsBucket, pageVisitKeys, endedMarkCodec{},
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", endedPageVisitsBucket, err)
	}
	return ledger, nil
}

func (ledger pageVisitLedger) isPending(tx *vault.Txn, key pageVisitKey) (bool, error) {
	_, pending, err := ledger.pending.Get(tx, key)
	return pending, err
}

func (ledger pageVisitLedger) add(
	tx *vault.Txn,
	pageVisit pendingpagevisit.PendingPageVisit,
	now time.Time,
) (bool, error) {
	key := pageVisitKeyOf(pageVisit)
	_, pending, err := ledger.pending.Get(tx, key)
	if err != nil || pending {
		return false, err
	}
	_, ended, err := ledger.ended.Get(tx, key)
	if err != nil || ended {
		return false, err
	}
	_, err = ledger.pending.Put(tx, key, pendingPageVisitRecord{
		due:   now.Unix(),
		depth: pageVisit.Depth,
		url:   pageVisit.URL,
	})
	return err == nil, err
}

func pageVisitKeyOf(pageVisit pendingpagevisit.PendingPageVisit) pageVisitKey {
	return pageVisitKey{crawlID: pageVisit.CrawlID, urlHash: urlhash.Of(pageVisit.URL)}
}

func (ledger pageVisitLedger) firstDueFrom(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	position duePosition,
) (dueEntry, bool, error) {
	return ledger.pendingByDue.First(
		tx,
		dueKeyParts.KeysWithFirstFrom(crawlID, position.due, position.urlHash),
	)
}

func (ledger pageVisitLedger) hasPending(tx *vault.Txn, crawlID crawlid.CrawlID) (bool, error) {
	amountPending, err := ledger.amountPendingOf(tx, crawlID)
	return amountPending > 0, err
}

func (ledger pageVisitLedger) forgetCrawl(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if _, err := ledger.pending.DeleteWithin(
		tx,
		pageVisitKeyParts.KeysWithFirst(crawlID),
	); err != nil {
		return fmt.Errorf("forget the pending page visits of crawl %s: %w", crawlID, err)
	}
	if _, err := ledger.ended.DeleteWithin(
		tx,
		pageVisitKeyParts.KeysWithFirst(crawlID),
	); err != nil {
		return fmt.Errorf("forget the ended page visits of crawl %s: %w", crawlID, err)
	}
	return nil
}

func (ledger pageVisitLedger) amountPendingOf(tx *vault.Txn, crawlID crawlid.CrawlID) (int, error) {
	return ledger.pendingOfCrawl.Get(tx, crawlID)
}

func (ledger pageVisitLedger) amountsPendingByHostOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	fn func(host string, amount int),
) error {
	return ledger.pendingOfHost.Scan(
		tx,
		crawlHostKeyParts.KeysWithFirst(crawlID),
		func(host crawlHost, amount int) (bool, error) {
			fn(host.host, amount)
			return true, nil
		},
	)
}

func (ledger pageVisitLedger) drainFrom(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	start urlhash.URLHash,
	amount int,
) (map[string]uint, int, error) {
	var drained []pageVisitKey
	amountsDrainedByHost := map[string]uint{}
	for _, pendingKeys := range []vault.KeyRange{
		pageVisitKeyParts.KeysWithFirst(crawlID).After(pageVisitKeyParts.Key(crawlID, start)),
		pageVisitKeyParts.KeysWithFirstBetweenSeconds(crawlID, urlhash.URLHash{}, start),
	} {
		if len(drained) == amount {
			break
		}
		if err := ledger.pending.Scan(
			tx,
			pendingKeys,
			func(key pageVisitKey, visit pendingPageVisitRecord) (bool, error) {
				drained = append(drained, key)
				amountsDrainedByHost[visit.url.Hostname()]++
				return len(drained) < amount, nil
			},
		); err != nil {
			return nil, 0, err
		}
	}
	for _, key := range drained {
		if _, err := ledger.pending.Delete(tx, key); err != nil {
			return nil, 0, err
		}
	}
	return amountsDrainedByHost, len(drained), nil
}

func (ledger pageVisitLedger) lease(
	tx *vault.Txn,
	key pageVisitKey,
	leasedUntil int64,
) (pendingPageVisitRecord, error) {
	visit, _, err := ledger.pending.Get(tx, key)
	if err != nil {
		return pendingPageVisitRecord{}, err
	}
	visit.due = leasedUntil
	_, err = ledger.pending.Put(tx, key, visit)
	return visit, err
}

func (ledger pageVisitLedger) amountPending(tx *vault.Txn) (int, error) {
	return ledger.pending.Len(tx)
}

func (ledger pageVisitLedger) endLeased(
	tx *vault.Txn,
	key pageVisitKey,
	leasedUntil int64,
) (bool, error) {
	_, leased, err := ledger.leasedVisitOf(tx, key, leasedUntil)
	if err != nil || !leased {
		return false, err
	}
	if _, err := ledger.pending.Delete(tx, key); err != nil {
		return false, err
	}
	_, err = ledger.ended.Put(tx, key, endedMark{})
	return err == nil, err
}

func (ledger pageVisitLedger) leasedVisitOf(
	tx *vault.Txn,
	key pageVisitKey,
	leasedUntil int64,
) (pendingPageVisitRecord, bool, error) {
	visit, pending, err := ledger.pending.Get(tx, key)
	if err != nil || !pending || visit.due != leasedUntil {
		return pendingPageVisitRecord{}, false, err
	}
	return visit, true, nil
}

func (ledger pageVisitLedger) returnLeased(
	tx *vault.Txn,
	key pageVisitKey,
	leasedUntil int64,
	due int64,
) (bool, error) {
	visit, leased, err := ledger.leasedVisitOf(tx, key, leasedUntil)
	if err != nil || !leased {
		return false, err
	}
	visit.due = due
	_, err = ledger.pending.Put(tx, key, visit)
	return err == nil, err
}

var pageVisitKeyParts = vault.PairKey(crawlid.KeyPart, urlhash.KeyPart)

var pageVisitKeys = pageVisitKeyParts.KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.urlHash },
	func(crawlID crawlid.CrawlID, urlHash urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, urlHash: urlHash}
	},
)

type pendingPageVisitWithKey struct {
	key   pageVisitKey
	visit pendingPageVisitRecord
}

type dueEntry struct {
	key pageVisitKey
	due int64
}

func (entry dueEntry) position() duePosition {
	return duePosition{due: entry.due, urlHash: entry.key.urlHash}
}

type duePosition struct {
	due     int64
	urlHash urlhash.URLHash
}

var earliestPosition = duePosition{due: math.MinInt64}

func (position duePosition) isBefore(other duePosition) bool {
	return position.due < other.due ||
		position.due == other.due && bytes.Compare(position.urlHash[:], other.urlHash[:]) < 0
}

var dueKeyParts = vault.TripleKey(
	crawlid.KeyPart,
	vault.IntegerKeyPart,
	urlhash.KeyPart,
)

var dueEntries = dueKeyParts.KeyLayoutFor(
	func(entry dueEntry) (crawlid.CrawlID, int64, urlhash.URLHash) {
		return entry.key.crawlID, entry.due, entry.key.urlHash
	},
	func(crawlID crawlid.CrawlID, due int64, urlHash urlhash.URLHash) dueEntry {
		return dueEntry{key: pageVisitKey{crawlID: crawlID, urlHash: urlHash}, due: due}
	},
)

type crawlHost struct {
	crawlID crawlid.CrawlID
	host    string
}

var crawlHostKeyParts = vault.PairKey(crawlid.KeyPart, vault.TextKeyPart)

var crawlHosts = crawlHostKeyParts.KeyLayoutFor(
	func(host crawlHost) (crawlid.CrawlID, string) { return host.crawlID, host.host },
	func(crawlID crawlid.CrawlID, host string) crawlHost {
		return crawlHost{crawlID: crawlID, host: host}
	},
)

type pendingPageVisitRecordCodec struct{}

func (pendingPageVisitRecordCodec) Encode(visit pendingPageVisitRecord) ([]byte, error) {
	address := visit.url.String()
	encoded := make([]byte, 0, 2*binary.MaxVarintLen64+len(address))
	encoded = binary.AppendVarint(encoded, visit.due)
	encoded = binary.AppendVarint(encoded, int64(visit.depth))
	return append(encoded, address...), nil
}

func (pendingPageVisitRecordCodec) Decode(stored []byte) (pendingPageVisitRecord, error) {
	due, read := binary.Varint(stored)
	if read <= 0 {
		return pendingPageVisitRecord{}, errPendingPageVisitRecordMalformed
	}
	rest := stored[read:]
	depth, read := binary.Varint(rest)
	if read <= 0 {
		return pendingPageVisitRecord{}, errPendingPageVisitRecordMalformed
	}
	url, err := canonicalurl.CanonicalURLOf(string(rest[read:]))
	if err != nil {
		return pendingPageVisitRecord{}, fmt.Errorf(
			"%w: %w",
			errPendingPageVisitRecordMalformed,
			err,
		)
	}
	return pendingPageVisitRecord{due: due, depth: int(depth), url: url}, nil
}

type endedMarkCodec struct{}

func (endedMarkCodec) Encode(endedMark) ([]byte, error) {
	return []byte{}, nil
}

func (endedMarkCodec) Decode([]byte) (endedMark, error) {
	return endedMark{}, nil
}
