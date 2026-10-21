// Package vault keeps the frontier in a vault: every page visit a crawl still owes,
// each due at a time. A worker takes a due visit for a lease, and the visit comes
// back to the frontier when the lease ends before the worker settles it. A visit
// already pending is not added twice, and the visits one call adds are added whole.
package vault

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlkeyparts"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

const (
	frontierName             = "YACY_CRAWL_FRONTIER"
	pageVisitsBucket         = "pending_page_visits"
	dueTimesBucket           = "pending_page_visits_by_due"
	dueTimesVersion          = "1"
	noDueVisitPause          = 500 * time.Millisecond
	pageVisitsPerPublication = 256
)

var pageVisitKeys = vault.PairKey(crawlkeyparts.CrawlID, crawlkeyparts.URLHash).KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.url },
	func(crawlID crawlid.CrawlID, url urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, url: url}
	},
)

var dueKeyParts = vault.TripleKey(
	vault.IntegerKeyPart,
	crawlkeyparts.CrawlID,
	crawlkeyparts.URLHash,
)

var dueEntries = dueKeyParts.KeyLayoutFor(
	func(entry dueEntry) (int64, crawlid.CrawlID, urlhash.URLHash) {
		return entry.due, entry.key.crawlID, entry.key.url
	},
	func(due int64, crawlID crawlid.CrawlID, url urlhash.URLHash) dueEntry {
		return dueEntry{due: due, key: pageVisitKey{crawlID: crawlID, url: url}}
	},
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type FrontierObserver interface {
	FrontierLeaseFailed(ctx context.Context, err error)
	FrontierCountFailed(ctx context.Context, err error)
	FrontierAcknowledgeFailed(ctx context.Context, identity string, err error)
	FrontierReturnFailed(
		ctx context.Context,
		identity string,
		delay time.Duration,
		err error,
	)
}

type PendingPageVisits struct {
	storage    *vault.Vault
	pageVisits *vault.Collection[pageVisitKey, pendingEntry]
	dueTimes   *vault.ValueIndex[dueEntry, dueEntry]
	lease      time.Duration
	clock      Clock
	observer   FrontierObserver
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	url     urlhash.URLHash
}

type dueEntry struct {
	due int64
	key pageVisitKey
}

func New(
	ctx context.Context,
	storage *vault.Vault,
	lease time.Duration,
	clock Clock,
	observer FrontierObserver,
) (*PendingPageVisits, error) {
	dueTimes, err := storage.RegisterValueIndex(
		dueTimesBucket, dueTimesVersion, dueEntries, func(entry dueEntry) dueEntry { return entry },
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", dueTimesBucket, err)
	}
	pageVisits, err := storage.RegisterProjectedCollection(
		ctx,
		pageVisitsBucket,
		pageVisitKeys,
		pendingEntryCodec{},
		vault.ProjectedAs(func(key pageVisitKey, entry pendingEntry) dueEntry {
			return dueEntry{due: entry.due, key: key}
		}, dueTimes),
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", pageVisitsBucket, err)
	}
	return &PendingPageVisits{
		storage:    storage,
		pageVisits: pageVisits,
		dueTimes:   dueTimes,
		lease:      lease,
		clock:      clock,
		observer:   observer,
	}, nil
}

func (visits *PendingPageVisits) Publish(
	ctx context.Context,
	pageVisits []pendingpagevisit.PendingPageVisit,
) error {
	published, err := publishedPageVisitsOf(pageVisits)
	if err != nil {
		return err
	}
	now := visits.clock.Now()
	for first := 0; first < len(published); first += pageVisitsPerPublication {
		publication := published[first:min(first+pageVisitsPerPublication, len(published))]
		if err := visits.storage.Update(ctx, func(tx *vault.Txn) error {
			return visits.addAbsent(tx, publication, now)
		}); err != nil {
			return fmt.Errorf("add %d pending page visits: %w", len(publication), err)
		}
	}
	return nil
}

type publishedPageVisit struct {
	key  pageVisitKey
	body []byte
}

func publishedPageVisitsOf(
	pageVisits []pendingpagevisit.PendingPageVisit,
) ([]publishedPageVisit, error) {
	published := make([]publishedPageVisit, 0, len(pageVisits))
	for _, pageVisit := range pageVisits {
		body, err := pendingpagevisit.MarshalPendingPageVisit(pageVisit)
		if err != nil {
			return nil, err
		}
		published = append(published, publishedPageVisit{
			key:  pageVisitKey{crawlID: pageVisit.CrawlID, url: urlhash.Of(pageVisit.URL)},
			body: body,
		})
	}
	return published, nil
}

func (visits *PendingPageVisits) addAbsent(
	tx *vault.Txn,
	publication []publishedPageVisit,
	now time.Time,
) error {
	for _, published := range publication {
		_, pending, err := visits.pageVisits.Get(tx, published.key)
		if err != nil {
			return err
		}
		if pending {
			continue
		}
		if _, err := visits.pageVisits.Put(tx, published.key, pendingEntry{
			due:       now.UnixMilli(),
			arrival:   now.UnixMilli(),
			messageID: newMessageID(),
			body:      published.body,
		}); err != nil {
			return err
		}
	}
	return nil
}

func newMessageID() messageID {
	var id messageID
	_, _ = rand.Read(id[:])
	return id
}

func (visits *PendingPageVisits) Next(ctx context.Context) (pullintake.PendingMessage, error) {
	for {
		message, leased := visits.leasedPageVisit(ctx)
		if leased {
			return message, nil
		}
		if err := visits.clock.Sleep(ctx, noDueVisitPause); err != nil {
			return nil, err
		}
	}
}

func (visits *PendingPageVisits) leasedPageVisit(
	ctx context.Context,
) (pendingPageVisitMessage, bool) {
	now := visits.clock.Now()
	due, err := visits.anyPageVisitDue(ctx, now)
	if err != nil {
		visits.observer.FrontierLeaseFailed(ctx, err)
		return pendingPageVisitMessage{}, false
	}
	if !due {
		return pendingPageVisitMessage{}, false
	}
	message, leased, err := visits.leaseDuePageVisit(ctx, now)
	if err != nil {
		visits.observer.FrontierLeaseFailed(ctx, err)
		return pendingPageVisitMessage{}, false
	}
	return message, leased
}

func (visits *PendingPageVisits) anyPageVisitDue(ctx context.Context, now time.Time) (bool, error) {
	var due bool
	if err := visits.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		_, due, err = visits.firstDueBy(tx, now)
		return err
	}); err != nil {
		return false, fmt.Errorf("look for a due pending page visit: %w", err)
	}
	return due, nil
}

func (visits *PendingPageVisits) firstDueBy(tx *vault.Txn, now time.Time) (dueEntry, bool, error) {
	var first dueEntry
	var found bool
	err := visits.dueTimes.Scan(
		tx,
		dueKeyParts.KeysThroughFirst(now.UnixMilli()),
		func(entry dueEntry) (bool, error) {
			first, found = entry, true
			return false, nil
		},
	)
	return first, found, err
}

func (visits *PendingPageVisits) leaseDuePageVisit(
	ctx context.Context,
	now time.Time,
) (pendingPageVisitMessage, bool, error) {
	var message pendingPageVisitMessage
	var leased bool
	if err := visits.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		message, leased, err = visits.leaseFirstDue(tx, now)
		return err
	}); err != nil {
		return pendingPageVisitMessage{}, false, fmt.Errorf(
			"lease a due pending page visit: %w",
			err,
		)
	}
	return message, leased, nil
}

func (visits *PendingPageVisits) leaseFirstDue(
	tx *vault.Txn,
	now time.Time,
) (pendingPageVisitMessage, bool, error) {
	first, due, err := visits.firstDueBy(tx, now)
	if err != nil || !due {
		return pendingPageVisitMessage{}, false, err
	}
	entry, pending, err := visits.pageVisits.Get(tx, first.key)
	if err != nil || !pending {
		return pendingPageVisitMessage{}, false, err
	}
	entry.due = now.Add(visits.lease).UnixMilli()
	if _, err := visits.pageVisits.Put(tx, first.key, entry); err != nil {
		return pendingPageVisitMessage{}, false, err
	}
	return pendingPageVisitMessage{
		visits:   visits,
		key:      first.key,
		identity: frontierName + "/" + hex.EncodeToString(entry.messageID[:]),
		arrival:  time.UnixMilli(entry.arrival),
		body:     entry.body,
	}, true, nil
}

func (visits *PendingPageVisits) AmountOfPendingPageVisits(ctx context.Context) (int64, bool) {
	var amount int
	if err := visits.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		amount, err = visits.pageVisits.Len(tx)
		return err
	}); err != nil {
		visits.observer.FrontierCountFailed(ctx, err)
		return 0, false
	}
	return int64(amount), true
}

func (visits *PendingPageVisits) acknowledge(
	ctx context.Context,
	key pageVisitKey,
	identity string,
) {
	if err := visits.storage.Update(ctx, func(tx *vault.Txn) error {
		_, err := visits.pageVisits.Delete(tx, key)
		return err
	}); err != nil {
		visits.observer.FrontierAcknowledgeFailed(ctx, identity, err)
	}
}

func (visits *PendingPageVisits) returnAfter(
	ctx context.Context,
	key pageVisitKey,
	identity string,
	delay time.Duration,
) {
	due := visits.clock.Now().Add(delay).UnixMilli()
	if err := visits.storage.Update(ctx, func(tx *vault.Txn) error {
		entry, pending, err := visits.pageVisits.Get(tx, key)
		if err != nil || !pending {
			return err
		}
		entry.due = due
		_, err = visits.pageVisits.Put(tx, key, entry)
		return err
	}); err != nil {
		visits.observer.FrontierReturnFailed(ctx, identity, delay, err)
	}
}
