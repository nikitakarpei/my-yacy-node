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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	pendingPageVisitsBucket        = "pending_page_visits"
	pendingPageVisitsByDueBucket   = "pending_page_visits_by_due"
	pendingPageVisitsOfCrawlBucket = "pending_page_visits_of_crawl"
	pendingPageVisitsOfHostBucket  = "pending_page_visits_of_host"
	endedPageVisitsBucket          = "ended_page_visits"
)

var errScheduledPageVisitMalformed = errors.New("scheduled page visit has a malformed field")

type pageVisitLedger struct {
	pending        *vault.Collection[pageVisitKey, scheduledPageVisit]
	pendingByDue   *vault.ValueIndex[scheduledPageVisitWithKey, dueEntry]
	pendingOfCrawl *vault.CountIndex[scheduledPageVisitWithKey, crawlid.CrawlID]
	pendingOfHost  *vault.CountIndex[scheduledPageVisitWithKey, crawlHost]
	ended          *vault.Collection[pageVisitKey, endedMark]
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	url     urlhash.URLHash
}

type scheduledPageVisit struct {
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
		func(scheduled scheduledPageVisitWithKey) dueEntry {
			return dueEntry{key: scheduled.key, due: scheduled.visit.due}
		},
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", pendingPageVisitsByDueBucket, err)
	}
	if ledger.pendingOfCrawl, err = storage.RegisterCountIndex(
		pendingPageVisitsOfCrawlBucket, frontierIndexesVersion,
		vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout(),
		func(scheduled scheduledPageVisitWithKey) crawlid.CrawlID { return scheduled.key.crawlID },
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", pendingPageVisitsOfCrawlBucket, err)
	}
	if ledger.pendingOfHost, err = storage.RegisterCountIndex(
		pendingPageVisitsOfHostBucket, frontierIndexesVersion, crawlHosts,
		func(scheduled scheduledPageVisitWithKey) crawlHost {
			return crawlHost{crawlID: scheduled.key.crawlID, host: scheduled.visit.url.Hostname()}
		},
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", pendingPageVisitsOfHostBucket, err)
	}
	if ledger.pending, err = storage.RegisterProjectedCollection(
		ctx, pendingPageVisitsBucket, pageVisitKeys, scheduledPageVisitCodec{},
		vault.ProjectedAs(
			func(key pageVisitKey, visit scheduledPageVisit) scheduledPageVisitWithKey {
				return scheduledPageVisitWithKey{key: key, visit: visit}
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
	_, err = ledger.pending.Put(tx, key, scheduledPageVisit{
		due:   now.Unix(),
		depth: pageVisit.Depth,
		url:   pageVisit.URL,
	})
	return err == nil, err
}

func pageVisitKeyOf(pageVisit pendingpagevisit.PendingPageVisit) pageVisitKey {
	return pageVisitKey{crawlID: pageVisit.CrawlID, url: urlhash.Of(pageVisit.URL)}
}

func (ledger pageVisitLedger) firstDueFrom(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	position duePosition,
) (dueEntry, bool, error) {
	return ledger.pendingByDue.First(
		tx,
		dueKeyParts.KeysWithFirstFrom(crawlID, position.due, position.url),
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
	drainedOfHosts := map[string]uint{}
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
			func(key pageVisitKey, visit scheduledPageVisit) (bool, error) {
				drained = append(drained, key)
				drainedOfHosts[visit.url.Hostname()]++
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
	return drainedOfHosts, len(drained), nil
}

func (ledger pageVisitLedger) lease(
	tx *vault.Txn,
	key pageVisitKey,
	leasedUntil int64,
) (scheduledPageVisit, error) {
	visit, _, err := ledger.pending.Get(tx, key)
	if err != nil {
		return scheduledPageVisit{}, err
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
) (scheduledPageVisit, bool, error) {
	visit, pending, err := ledger.pending.Get(tx, key)
	if err != nil || !pending || visit.due != leasedUntil {
		return scheduledPageVisit{}, false, err
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

var pageVisitKeyParts = vault.PairKey(vaultkeyparts.CrawlID, vaultkeyparts.URLHash)

var pageVisitKeys = pageVisitKeyParts.KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.url },
	func(crawlID crawlid.CrawlID, url urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, url: url}
	},
)

type scheduledPageVisitWithKey struct {
	key   pageVisitKey
	visit scheduledPageVisit
}

type dueEntry struct {
	key pageVisitKey
	due int64
}

func (entry dueEntry) position() duePosition {
	return duePosition{due: entry.due, url: entry.key.url}
}

type duePosition struct {
	due int64
	url urlhash.URLHash
}

var earliestPosition = duePosition{due: math.MinInt64}

func (position duePosition) isBefore(other duePosition) bool {
	return position.due < other.due ||
		position.due == other.due && bytes.Compare(position.url[:], other.url[:]) < 0
}

var dueKeyParts = vault.TripleKey(
	vaultkeyparts.CrawlID,
	vault.IntegerKeyPart,
	vaultkeyparts.URLHash,
)

var dueEntries = dueKeyParts.KeyLayoutFor(
	func(entry dueEntry) (crawlid.CrawlID, int64, urlhash.URLHash) {
		return entry.key.crawlID, entry.due, entry.key.url
	},
	func(crawlID crawlid.CrawlID, due int64, url urlhash.URLHash) dueEntry {
		return dueEntry{key: pageVisitKey{crawlID: crawlID, url: url}, due: due}
	},
)

type crawlHost struct {
	crawlID crawlid.CrawlID
	host    string
}

var crawlHostKeyParts = vault.PairKey(vaultkeyparts.CrawlID, vault.TextKeyPart)

var crawlHosts = crawlHostKeyParts.KeyLayoutFor(
	func(host crawlHost) (crawlid.CrawlID, string) { return host.crawlID, host.host },
	func(crawlID crawlid.CrawlID, host string) crawlHost {
		return crawlHost{crawlID: crawlID, host: host}
	},
)

type scheduledPageVisitCodec struct{}

func (scheduledPageVisitCodec) Encode(visit scheduledPageVisit) ([]byte, error) {
	address := visit.url.String()
	encoded := make([]byte, 0, 2*binary.MaxVarintLen64+len(address))
	encoded = binary.AppendVarint(encoded, visit.due)
	encoded = binary.AppendVarint(encoded, int64(visit.depth))
	return append(encoded, address...), nil
}

func (scheduledPageVisitCodec) Decode(stored []byte) (scheduledPageVisit, error) {
	due, read := binary.Varint(stored)
	if read <= 0 {
		return scheduledPageVisit{}, errScheduledPageVisitMalformed
	}
	rest := stored[read:]
	depth, read := binary.Varint(rest)
	if read <= 0 {
		return scheduledPageVisit{}, errScheduledPageVisitMalformed
	}
	url, err := canonicalurl.CanonicalURLOf(string(rest[read:]))
	if err != nil {
		return scheduledPageVisit{}, fmt.Errorf("%w: %w", errScheduledPageVisitMalformed, err)
	}
	return scheduledPageVisit{due: due, depth: int(depth), url: url}, nil
}

type endedMarkCodec struct{}

func (endedMarkCodec) Encode(endedMark) ([]byte, error) {
	return []byte{}, nil
}

func (endedMarkCodec) Decode([]byte) (endedMark, error) {
	return endedMark{}, nil
}
