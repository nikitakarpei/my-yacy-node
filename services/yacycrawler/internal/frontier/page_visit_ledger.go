package frontier

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	owedPageVisitsBucket        = "owed_page_visits"
	owedPageVisitsByDueBucket   = "owed_page_visits_by_due"
	owedPageVisitsOfCrawlBucket = "owed_page_visits_of_crawl"
	owedPageVisitsOfHostBucket  = "owed_page_visits_of_host"
	settledPageVisitsBucket     = "settled_page_visits"
)

var errOwedPageVisitMalformed = errors.New("owed page visit has a malformed field")

type pageVisitLedger struct {
	owed        *vault.Collection[pageVisitKey, owedPageVisit]
	owedByDue   *vault.ValueIndex[owedPageVisitForm, dueEntry]
	owedOfCrawl *vault.CountIndex[owedPageVisitForm, crawlid.CrawlID]
	owedOfHost  *vault.CountIndex[owedPageVisitForm, crawlHost]
	settled     *vault.Collection[pageVisitKey, settledMark]
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	url     urlhash.URLHash
}

type owedPageVisit struct {
	due   int64
	depth int
	url   canonicalurl.CanonicalURL
}

type settledMark struct{}

func registerPageVisitLedger(ctx context.Context, storage *vault.Vault) (pageVisitLedger, error) {
	var ledger pageVisitLedger
	var err error
	if ledger.owedByDue, err = storage.RegisterValueIndex(
		owedPageVisitsByDueBucket,
		frontierIndexesVersion,
		dueEntries,
		func(form owedPageVisitForm) dueEntry { return dueEntry{key: form.key, due: form.visit.due} },
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", owedPageVisitsByDueBucket, err)
	}
	if ledger.owedOfCrawl, err = storage.RegisterCountIndex(
		owedPageVisitsOfCrawlBucket, frontierIndexesVersion,
		vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout(),
		func(form owedPageVisitForm) crawlid.CrawlID { return form.key.crawlID },
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", owedPageVisitsOfCrawlBucket, err)
	}
	if ledger.owedOfHost, err = storage.RegisterCountIndex(
		owedPageVisitsOfHostBucket, frontierIndexesVersion, crawlHosts,
		func(form owedPageVisitForm) crawlHost {
			return crawlHost{crawlID: form.key.crawlID, host: form.visit.url.Hostname()}
		},
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", owedPageVisitsOfHostBucket, err)
	}
	if ledger.owed, err = storage.RegisterProjectedCollection(
		ctx, owedPageVisitsBucket, pageVisitKeys, owedPageVisitCodec{},
		vault.ProjectedAs(
			func(key pageVisitKey, visit owedPageVisit) owedPageVisitForm {
				return owedPageVisitForm{key: key, visit: visit}
			},
			ledger.owedByDue, ledger.owedOfCrawl, ledger.owedOfHost,
		),
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", owedPageVisitsBucket, err)
	}
	if ledger.settled, err = storage.RegisterCollection(
		settledPageVisitsBucket, pageVisitKeys, settledMarkCodec{},
	); err != nil {
		return pageVisitLedger{}, fmt.Errorf("register %s: %w", settledPageVisitsBucket, err)
	}
	return ledger, nil
}

func (ledger pageVisitLedger) owe(
	tx *vault.Txn,
	pageVisit pendingpagevisit.PendingPageVisit,
	now time.Time,
) (bool, error) {
	key := pageVisitKeyOf(pageVisit)
	_, owed, err := ledger.owed.Get(tx, key)
	if err != nil || owed {
		return false, err
	}
	_, settled, err := ledger.settled.Get(tx, key)
	if err != nil || settled {
		return false, err
	}
	_, err = ledger.owed.Put(tx, key, owedPageVisit{
		due:   now.Unix(),
		depth: pageVisit.Depth,
		url:   pageVisit.URL,
	})
	return err == nil, err
}

func pageVisitKeyOf(pageVisit pendingpagevisit.PendingPageVisit) pageVisitKey {
	return pageVisitKey{crawlID: pageVisit.CrawlID, url: urlhash.Of(pageVisit.URL)}
}

func (ledger pageVisitLedger) firstDueOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (dueEntry, bool, error) {
	return ledger.owedByDue.First(tx, dueKeyParts.KeysWithFirst(crawlID))
}

func (ledger pageVisitLedger) owesAny(tx *vault.Txn, crawlID crawlid.CrawlID) (bool, error) {
	_, owes, err := ledger.firstDueOf(tx, crawlID)
	return owes, err
}

func (ledger pageVisitLedger) forgetCrawl(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if _, err := ledger.owed.DeleteWithin(
		tx,
		pageVisitKeyParts.KeysWithFirst(crawlID),
	); err != nil {
		return fmt.Errorf("forget the owed page visits of crawl %s: %w", crawlID, err)
	}
	if _, err := ledger.settled.DeleteWithin(
		tx,
		pageVisitKeyParts.KeysWithFirst(crawlID),
	); err != nil {
		return fmt.Errorf("forget the settled page visits of crawl %s: %w", crawlID, err)
	}
	return nil
}

func (ledger pageVisitLedger) amountOwedBy(tx *vault.Txn, crawlID crawlid.CrawlID) (int, error) {
	return ledger.owedOfCrawl.Get(tx, crawlID)
}

func (ledger pageVisitLedger) amountsOwedByHostsOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	fn func(host string, amount int),
) error {
	return ledger.owedOfHost.Scan(
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
	for _, owedKeys := range []vault.KeyRange{
		pageVisitKeyParts.KeysWithFirst(crawlID).After(pageVisitKeyParts.Key(crawlID, start)),
		pageVisitKeyParts.KeysWithFirstBetweenSeconds(crawlID, urlhash.URLHash{}, start),
	} {
		if len(drained) == amount {
			break
		}
		if err := ledger.owed.Scan(
			tx,
			owedKeys,
			func(key pageVisitKey, visit owedPageVisit) (bool, error) {
				drained = append(drained, key)
				drainedOfHosts[visit.url.Hostname()]++
				return len(drained) < amount, nil
			},
		); err != nil {
			return nil, 0, err
		}
	}
	for _, key := range drained {
		if _, err := ledger.owed.Delete(tx, key); err != nil {
			return nil, 0, err
		}
	}
	return drainedOfHosts, len(drained), nil
}

func (ledger pageVisitLedger) lease(
	tx *vault.Txn,
	key pageVisitKey,
	leasedUntil int64,
) (owedPageVisit, error) {
	visit, _, err := ledger.owed.Get(tx, key)
	if err != nil {
		return owedPageVisit{}, err
	}
	visit.due = leasedUntil
	_, err = ledger.owed.Put(tx, key, visit)
	return visit, err
}

func (ledger pageVisitLedger) amountOwed(tx *vault.Txn) (int, error) {
	return ledger.owed.Len(tx)
}

func (ledger pageVisitLedger) settleLeased(
	tx *vault.Txn,
	key pageVisitKey,
	leasedUntil int64,
) (bool, error) {
	_, leased, err := ledger.leasedVisitOf(tx, key, leasedUntil)
	if err != nil || !leased {
		return false, err
	}
	if _, err := ledger.owed.Delete(tx, key); err != nil {
		return false, err
	}
	_, err = ledger.settled.Put(tx, key, settledMark{})
	return err == nil, err
}

func (ledger pageVisitLedger) leasedVisitOf(
	tx *vault.Txn,
	key pageVisitKey,
	leasedUntil int64,
) (owedPageVisit, bool, error) {
	visit, owed, err := ledger.owed.Get(tx, key)
	if err != nil || !owed || visit.due != leasedUntil {
		return owedPageVisit{}, false, err
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
	_, err = ledger.owed.Put(tx, key, visit)
	return err == nil, err
}

var pageVisitKeyParts = vault.PairKey(vaultkeyparts.CrawlID, vaultkeyparts.URLHash)

var pageVisitKeys = pageVisitKeyParts.KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.url },
	func(crawlID crawlid.CrawlID, url urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, url: url}
	},
)

type owedPageVisitForm struct {
	key   pageVisitKey
	visit owedPageVisit
}

type dueEntry struct {
	key pageVisitKey
	due int64
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

type owedPageVisitCodec struct{}

func (owedPageVisitCodec) Encode(visit owedPageVisit) ([]byte, error) {
	address := visit.url.String()
	encoded := make([]byte, 0, 2*binary.MaxVarintLen64+len(address))
	encoded = binary.AppendVarint(encoded, visit.due)
	encoded = binary.AppendVarint(encoded, int64(visit.depth))
	return append(encoded, address...), nil
}

func (owedPageVisitCodec) Decode(stored []byte) (owedPageVisit, error) {
	due, read := binary.Varint(stored)
	if read <= 0 {
		return owedPageVisit{}, errOwedPageVisitMalformed
	}
	rest := stored[read:]
	depth, read := binary.Varint(rest)
	if read <= 0 {
		return owedPageVisit{}, errOwedPageVisitMalformed
	}
	url, err := canonicalurl.CanonicalURLOf(string(rest[read:]))
	if err != nil {
		return owedPageVisit{}, fmt.Errorf("%w: %w", errOwedPageVisitMalformed, err)
	}
	return owedPageVisit{due: due, depth: int(depth), url: url}, nil
}

type settledMarkCodec struct{}

func (settledMarkCodec) Encode(settledMark) ([]byte, error) {
	return []byte{}, nil
}

func (settledMarkCodec) Decode([]byte) (settledMark, error) {
	return settledMark{}, nil
}
