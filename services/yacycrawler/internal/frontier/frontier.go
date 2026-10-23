// Package frontier keeps every pending page visit of a crawl, in a queue of its crawl, each
// due at a time, and a mark of every visit that ended. A worker leases the earliest due
// visit of a crawl neither held nor closed, and the visit comes back when the lease expires
// before the worker ends it. A visit pending or ended before is not added again, and a
// closed or cancelled crawl takes none. A cancelled crawl's pending visits are dropped in
// batches and reported as cancelled. A pause holds every page visit.
package frontier

import (
	"context"
	cryptorand "crypto/rand"
	"fmt"
	"maps"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

const (
	frontierIndexesVersion      = "1"
	noDueVisitPause             = 500 * time.Millisecond
	noCancelledCrawlPause       = 500 * time.Millisecond
	pageVisitsDrainedAtOnce     = 100
	cancellingCrawlsPickedAmong = 8
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type FrontierObserver interface {
	FrontierLeaseFailed(ctx context.Context, err error)
	FrontierDrainFailed(ctx context.Context, err error)
	FrontierCountFailed(ctx context.Context, err error)
	FrontierEndFailed(
		ctx context.Context,
		pageVisit pendingpagevisit.PendingPageVisit,
		err error,
	)
	FrontierReturnFailed(
		ctx context.Context,
		pageVisit pendingpagevisit.PendingPageVisit,
		delay time.Duration,
		err error,
	)
	FrontierLeaseOutdated(ctx context.Context, pageVisit pendingpagevisit.PendingPageVisit)
}

type CancelledPageVisits interface {
	PageVisitsCancelled(tx *vault.Txn, crawlID crawlid.CrawlID, host string, amount uint) error
}

type Frontier struct {
	storage    *vault.Vault
	cancelled  CancelledPageVisits
	ledger     pageVisitLedger
	leases     pageVisitLeases
	crawlMarks crawlMarks
	pause      pageVisitPause
	clock      Clock
	observer   FrontierObserver
}

//nolint:revive // a frontier names the vault, the ports it reports to, its lease, clock and observer
func Open(
	ctx context.Context,
	storage *vault.Vault,
	cancelled CancelledPageVisits,
	lease time.Duration,
	clock Clock,
	observer FrontierObserver,
) (*Frontier, error) {
	frontier := &Frontier{
		storage:   storage,
		cancelled: cancelled,
		clock:     clock,
		observer:  observer,
	}
	var err error
	if frontier.ledger, err = registerPageVisitLedger(ctx, storage); err != nil {
		return nil, err
	}
	heads, err := registerCrawlHeads(ctx, storage)
	if err != nil {
		return nil, err
	}
	if frontier.crawlMarks, err = registerCrawlMarks(ctx, storage); err != nil {
		return nil, err
	}
	if frontier.pause, err = registerPageVisitPause(storage); err != nil {
		return nil, err
	}
	frontier.leases = pageVisitLeases{
		storage:  storage,
		ledger:   frontier.ledger,
		heads:    heads,
		pause:    frontier.pause,
		lease:    lease,
		clock:    clock,
		observer: observer,
	}
	return frontier, nil
}

func (frontier *Frontier) Add(tx *vault.Txn, pageVisits []pendingpagevisit.PendingPageVisit) error {
	now := frontier.clock.Now()
	for crawlID, pageVisitsOfCrawl := range pageVisitsByCrawl(pageVisits) {
		if err := frontier.addToCrawl(tx, crawlID, pageVisitsOfCrawl, now); err != nil {
			return fmt.Errorf("add page visits of crawl %s: %w", crawlID, err)
		}
	}
	return nil
}

func pageVisitsByCrawl(
	pageVisits []pendingpagevisit.PendingPageVisit,
) map[crawlid.CrawlID][]pendingpagevisit.PendingPageVisit {
	byCrawl := map[crawlid.CrawlID][]pendingpagevisit.PendingPageVisit{}
	for _, pageVisit := range pageVisits {
		byCrawl[pageVisit.CrawlID] = append(byCrawl[pageVisit.CrawlID], pageVisit)
	}
	return byCrawl
}

func (frontier *Frontier) addToCrawl(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	pageVisits []pendingpagevisit.PendingPageVisit,
	now time.Time,
) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || !mark.takesPageVisits() {
		return err
	}
	addedAny := false
	for _, pageVisit := range pageVisits {
		addedNow, err := frontier.ledger.add(tx, pageVisit, now)
		if err != nil {
			return err
		}
		addedAny = addedAny || addedNow
	}
	if !addedAny || mark == markHeld {
		return nil
	}
	return frontier.leases.scheduleNoLaterThan(tx, crawlID, now)
}

func (frontier *Frontier) Hold(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || mark != unmarked {
		return err
	}
	if err := frontier.crawlMarks.mark(tx, crawlID, markHeld); err != nil {
		return err
	}
	return frontier.leases.unschedule(tx, crawlID)
}

func (frontier *Frontier) Release(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || mark != markHeld {
		return err
	}
	if err := frontier.crawlMarks.unmark(tx, crawlID); err != nil {
		return err
	}
	return frontier.leases.reschedule(tx, crawlID)
}

func (frontier *Frontier) Close(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || !mark.takesPageVisits() {
		return err
	}
	if err := frontier.crawlMarks.mark(tx, crawlID, markClosed); err != nil {
		return err
	}
	return frontier.leases.unschedule(tx, crawlID)
}

func (frontier *Frontier) Cancel(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || !mark.takesPageVisits() {
		return err
	}
	pending, err := frontier.ledger.hasPending(tx, crawlID)
	if err != nil {
		return err
	}
	cancellingMark := markClosed
	if pending {
		cancellingMark = markCancelling
	}
	if err := frontier.crawlMarks.mark(tx, crawlID, cancellingMark); err != nil {
		return err
	}
	return frontier.leases.unschedule(tx, crawlID)
}

func (frontier *Frontier) ForgetCrawl(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if err := frontier.ledger.forgetCrawl(tx, crawlID); err != nil {
		return err
	}
	if err := frontier.leases.unschedule(tx, crawlID); err != nil {
		return err
	}
	return frontier.crawlMarks.unmark(tx, crawlID)
}

func (frontier *Frontier) AmountPendingOf(tx *vault.Txn, crawlID crawlid.CrawlID) (int, error) {
	return frontier.ledger.amountPendingOf(tx, crawlID)
}

func (frontier *Frontier) AmountsPendingByHostOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	fn func(host string, amount int),
) error {
	return frontier.ledger.amountsPendingByHostOf(tx, crawlID, fn)
}

func (frontier *Frontier) Next(ctx context.Context) (pendingpagevisit.LeasedPageVisit, error) {
	return frontier.leases.next(ctx)
}

func (frontier *Frontier) End(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
) {
	frontier.leases.end(ctx, leasedPageVisit)
}

func (frontier *Frontier) Return(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	delay time.Duration,
) {
	frontier.leases.returnAfter(ctx, leasedPageVisit, delay)
}

func (frontier *Frontier) DrainCancelledCrawls(ctx context.Context) error {
	for {
		if frontier.drainNextBatch(ctx) {
			continue
		}
		if err := frontier.clock.Sleep(ctx, noCancelledCrawlPause); err != nil {
			return nil //nolint:nilerr // a clock only stops sleeping early when the drain is to end
		}
	}
}

func (frontier *Frontier) drainNextBatch(ctx context.Context) bool {
	crawlID, cancelling, err := frontier.pickedCancellingCrawl(ctx)
	if err != nil {
		frontier.observer.FrontierDrainFailed(ctx, err)
		return false
	}
	if !cancelling {
		return false
	}
	start := randomURLHash()
	if err := frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		return frontier.drainBatchOf(tx, crawlID, start)
	}); err != nil {
		frontier.observer.FrontierDrainFailed(ctx, fmt.Errorf("drain crawl %s: %w", crawlID, err))
		return false
	}
	return true
}

func (frontier *Frontier) pickedCancellingCrawl(
	ctx context.Context,
) (crawlid.CrawlID, bool, error) {
	var cancelling []crawlid.CrawlID
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		cancelling, err = frontier.crawlMarks.firstCancellingCrawls(
			tx, cancellingCrawlsPickedAmong,
		)
		return err
	}); err != nil {
		return crawlid.CrawlID{}, false, fmt.Errorf("look for a crawl to drain: %w", err)
	}
	if len(cancelling) == 0 {
		return crawlid.CrawlID{}, false, nil
	}
	return cancelling[rand.IntN(len(cancelling))], true, nil //nolint:gosec // spreads processes, no secret
}

func randomURLHash() urlhash.URLHash {
	var start urlhash.URLHash
	_, _ = cryptorand.Read(start[:])
	return start
}

func (frontier *Frontier) drainBatchOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	start urlhash.URLHash,
) error {
	drainedOfHosts, amountDrained, err := frontier.ledger.drainFrom(
		tx, crawlID, start, pageVisitsDrainedAtOnce,
	)
	if err != nil {
		return err
	}
	for _, host := range slices.Sorted(maps.Keys(drainedOfHosts)) {
		if err := frontier.cancelled.PageVisitsCancelled(
			tx, crawlID, host, drainedOfHosts[host],
		); err != nil {
			return err
		}
	}
	if amountDrained == pageVisitsDrainedAtOnce {
		return nil
	}
	return frontier.closeDrained(tx, crawlID)
}

func (frontier *Frontier) closeDrained(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || mark != markCancelling {
		return err
	}
	return frontier.crawlMarks.mark(tx, crawlID, markClosed)
}

func (frontier *Frontier) AmountOfPendingPageVisits(ctx context.Context) (int64, bool) {
	var amount int
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		amount, err = frontier.ledger.amountPending(tx)
		return err
	}); err != nil {
		frontier.observer.FrontierCountFailed(ctx, err)
		return 0, false
	}
	return int64(amount), true
}

func (frontier *Frontier) Pause(ctx context.Context) error {
	if err := frontier.storage.Update(ctx, frontier.pause.turnOn); err != nil {
		return fmt.Errorf("pause every page visit: %w", err)
	}
	return nil
}

func (frontier *Frontier) Resume(ctx context.Context) error {
	if err := frontier.storage.Update(ctx, frontier.pause.turnOff); err != nil {
		return fmt.Errorf("resume every page visit: %w", err)
	}
	return nil
}

func (frontier *Frontier) IsPaused(ctx context.Context) (bool, error) {
	var paused bool
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		paused, err = frontier.pause.isOn(tx)
		return err
	}); err != nil {
		return false, fmt.Errorf("read the pause of every page visit: %w", err)
	}
	return paused, nil
}
