// Package frontier keeps every page visit a crawl still owes, each due at a time. A
// worker takes a due visit for a lease, and the visit comes back to the frontier when the
// lease ends before the worker settles it. A visit already owed is not added twice, and
// the visits one transaction owes are owed whole.
package frontier

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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

const (
	frontierName    = "YACY_CRAWL_FRONTIER"
	noDueVisitPause = 500 * time.Millisecond
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

type Frontier struct {
	storage    *vault.Vault
	pageVisits *vault.Collection[pageVisitKey, pendingEntry]
	dueTimes   *vault.ValueIndex[dueEntry, dueEntry]
	lease      time.Duration
	clock      Clock
	observer   FrontierObserver
}

func Open(
	ctx context.Context,
	storage *vault.Vault,
	lease time.Duration,
	clock Clock,
	observer FrontierObserver,
) (*Frontier, error) {
	pageVisits, dueTimes, err := registerFrontier(ctx, storage)
	if err != nil {
		return nil, err
	}
	return &Frontier{
		storage:    storage,
		pageVisits: pageVisits,
		dueTimes:   dueTimes,
		lease:      lease,
		clock:      clock,
		observer:   observer,
	}, nil
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	url     urlhash.URLHash
}

type dueEntry struct {
	due int64
	key pageVisitKey
}

type messageID [messageIDBytes]byte

type pendingEntry struct {
	due       int64
	arrival   int64
	messageID messageID
	body      []byte
}

func (frontier *Frontier) Owe(
	tx *vault.Txn,
	pageVisits []pendingpagevisit.PendingPageVisit,
) error {
	owed, err := owedPageVisitsOf(pageVisits)
	if err != nil {
		return err
	}
	if err := frontier.addAbsent(tx, owed, frontier.clock.Now()); err != nil {
		return fmt.Errorf("owe %d page visits: %w", len(owed), err)
	}
	return nil
}

type owedPageVisit struct {
	key  pageVisitKey
	body []byte
}

func owedPageVisitsOf(
	pageVisits []pendingpagevisit.PendingPageVisit,
) ([]owedPageVisit, error) {
	owed := make([]owedPageVisit, 0, len(pageVisits))
	for _, pageVisit := range pageVisits {
		body, err := pendingpagevisit.MarshalPendingPageVisit(pageVisit)
		if err != nil {
			return nil, err
		}
		owed = append(owed, owedPageVisit{
			key:  pageVisitKey{crawlID: pageVisit.CrawlID, url: urlhash.Of(pageVisit.URL)},
			body: body,
		})
	}
	return owed, nil
}

func (frontier *Frontier) addAbsent(
	tx *vault.Txn,
	owed []owedPageVisit,
	now time.Time,
) error {
	for _, visit := range owed {
		_, pending, err := frontier.pageVisits.Get(tx, visit.key)
		if err != nil {
			return err
		}
		if pending {
			continue
		}
		if _, err := frontier.pageVisits.Put(tx, visit.key, pendingEntry{
			due:       now.Unix(),
			arrival:   now.Unix(),
			messageID: newMessageID(),
			body:      visit.body,
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

func (frontier *Frontier) Next(ctx context.Context) (pullintake.PendingMessage, error) {
	for {
		message, leased := frontier.leaseNextPageVisit(ctx)
		if leased {
			return message, nil
		}
		if err := frontier.clock.Sleep(ctx, noDueVisitPause); err != nil {
			return nil, err
		}
	}
}

func (frontier *Frontier) leaseNextPageVisit(
	ctx context.Context,
) (pendingPageVisitMessage, bool) {
	now := frontier.clock.Now()
	due, err := frontier.anyPageVisitDue(ctx, now)
	if err != nil {
		frontier.observer.FrontierLeaseFailed(ctx, err)
		return pendingPageVisitMessage{}, false
	}
	if !due {
		return pendingPageVisitMessage{}, false
	}
	message, leased, err := frontier.leaseDuePageVisit(ctx, now)
	if err != nil {
		frontier.observer.FrontierLeaseFailed(ctx, err)
		return pendingPageVisitMessage{}, false
	}
	return message, leased
}

func (frontier *Frontier) anyPageVisitDue(ctx context.Context, now time.Time) (bool, error) {
	var due bool
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		_, due, err = frontier.firstDueBy(tx, now)
		return err
	}); err != nil {
		return false, fmt.Errorf("look for a due pending page visit: %w", err)
	}
	return due, nil
}

func (frontier *Frontier) firstDueBy(tx *vault.Txn, now time.Time) (dueEntry, bool, error) {
	var first dueEntry
	var found bool
	err := frontier.dueTimes.Scan(
		tx,
		dueKeyParts.KeysThroughFirst(now.Unix()),
		func(entry dueEntry) (bool, error) {
			first, found = entry, true
			return false, nil
		},
	)
	return first, found, err
}

func (frontier *Frontier) leaseDuePageVisit(
	ctx context.Context,
	now time.Time,
) (pendingPageVisitMessage, bool, error) {
	var message pendingPageVisitMessage
	var leased bool
	if err := frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		message, leased, err = frontier.leaseFirstDue(tx, now)
		return err
	}); err != nil {
		return pendingPageVisitMessage{}, false, fmt.Errorf(
			"lease a due pending page visit: %w",
			err,
		)
	}
	return message, leased, nil
}

func (frontier *Frontier) leaseFirstDue(
	tx *vault.Txn,
	now time.Time,
) (pendingPageVisitMessage, bool, error) {
	first, due, err := frontier.firstDueBy(tx, now)
	if err != nil || !due {
		return pendingPageVisitMessage{}, false, err
	}
	entry, pending, err := frontier.pageVisits.Get(tx, first.key)
	if err != nil || !pending {
		return pendingPageVisitMessage{}, false, err
	}
	entry.due = secondsAtOrAfter(now.Add(frontier.lease))
	if _, err := frontier.pageVisits.Put(tx, first.key, entry); err != nil {
		return pendingPageVisitMessage{}, false, err
	}
	return pendingPageVisitMessage{
		frontier: frontier,
		key:      first.key,
		identity: frontierName + "/" + hex.EncodeToString(entry.messageID[:]),
		arrival:  time.Unix(entry.arrival, 0),
		body:     entry.body,
	}, true, nil
}

func secondsAtOrAfter(at time.Time) int64 {
	seconds := at.Unix()
	if at.Nanosecond() > 0 {
		seconds++
	}
	return seconds
}

func (frontier *Frontier) AmountOfPendingPageVisits(ctx context.Context) (int64, bool) {
	var amount int
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		amount, err = frontier.pageVisits.Len(tx)
		return err
	}); err != nil {
		frontier.observer.FrontierCountFailed(ctx, err)
		return 0, false
	}
	return int64(amount), true
}

func (frontier *Frontier) acknowledge(
	ctx context.Context,
	key pageVisitKey,
	identity string,
) {
	if err := frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		_, err := frontier.pageVisits.Delete(tx, key)
		return err
	}); err != nil {
		frontier.observer.FrontierAcknowledgeFailed(ctx, identity, err)
	}
}

func (frontier *Frontier) returnAfter(
	ctx context.Context,
	key pageVisitKey,
	identity string,
	delay time.Duration,
) {
	due := secondsAtOrAfter(frontier.clock.Now().Add(delay))
	if err := frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		entry, pending, err := frontier.pageVisits.Get(tx, key)
		if err != nil || !pending {
			return err
		}
		entry.due = due
		_, err = frontier.pageVisits.Put(tx, key, entry)
		return err
	}); err != nil {
		frontier.observer.FrontierReturnFailed(ctx, identity, delay, err)
	}
}
