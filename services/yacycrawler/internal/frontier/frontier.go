// Package frontier keeps the pending page visits of each crawl in a queue, each due at a
// time, and a mark of every visit that ended. A worker leases the earliest due visit of a
// queue neither paused nor draining; it comes back when the lease expires first. A visit
// pending or ended before is not added again, nor what a visit leads to once it is not
// pending. A draining queue takes none and drops its visits in batches. A pause holds all.
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
	frontierIndexesVersion    = "1"
	noDueVisitPause           = 500 * time.Millisecond
	noDrainingQueuePause      = 500 * time.Millisecond
	pageVisitsDrainedAtOnce   = 100
	drainingQueuesPickedAmong = 8
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

type DrainedPageVisits interface {
	PageVisitsDrained(tx *vault.Txn, crawlID crawlid.CrawlID, host string, amount uint) error
}

type Frontier struct {
	storage     *vault.Vault
	drained     DrainedPageVisits
	ledger      pageVisitLedger
	leases      pageVisitLeases
	queueStates queueStates
	pause       pageVisitPause
	clock       Clock
	observer    FrontierObserver
}

//nolint:revive // a frontier names the vault, the ports it reports to, its lease, clock and observer
func Open(
	ctx context.Context,
	storage *vault.Vault,
	drained DrainedPageVisits,
	lease time.Duration,
	clock Clock,
	observer FrontierObserver,
) (*Frontier, error) {
	frontier := &Frontier{
		storage:  storage,
		drained:  drained,
		clock:    clock,
		observer: observer,
	}
	var err error
	if frontier.ledger, err = registerPageVisitLedger(ctx, storage); err != nil {
		return nil, err
	}
	heads, err := registerCrawlHeads(ctx, storage)
	if err != nil {
		return nil, err
	}
	if frontier.queueStates, err = registerQueueStates(ctx, storage); err != nil {
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

func (frontier *Frontier) AddSeeds(
	tx *vault.Txn,
	seedPageVisits []pendingpagevisit.PendingPageVisit,
) error {
	return frontier.add(tx, seedPageVisits)
}

func (frontier *Frontier) AddNextOf(
	tx *vault.Txn,
	pageVisit pendingpagevisit.PendingPageVisit,
	nextPageVisits []pendingpagevisit.PendingPageVisit,
) error {
	pending, err := frontier.ledger.isPending(tx, pageVisitKeyOf(pageVisit))
	if err != nil || !pending {
		return err
	}
	return frontier.add(tx, nextPageVisits)
}

func (frontier *Frontier) add(tx *vault.Txn, pageVisits []pendingpagevisit.PendingPageVisit) error {
	now := frontier.clock.Now()
	for crawlID, pageVisitsOfCrawl := range pageVisitsByCrawl(pageVisits) {
		if err := frontier.addToQueueOf(tx, crawlID, pageVisitsOfCrawl, now); err != nil {
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

func (frontier *Frontier) addToQueueOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	pageVisits []pendingpagevisit.PendingPageVisit,
	now time.Time,
) error {
	state, err := frontier.queueStates.stateOf(tx, crawlID)
	if err != nil || !state.takesPageVisits() {
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
	if !addedAny || state == queuePaused {
		return nil
	}
	return frontier.leases.startFrom(tx, crawlID, now)
}

func (frontier *Frontier) PauseQueueOf(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	state, err := frontier.queueStates.stateOf(tx, crawlID)
	if err != nil || state != queueOpen {
		return err
	}
	if err := frontier.queueStates.put(tx, crawlID, queuePaused); err != nil {
		return err
	}
	return frontier.leases.stop(tx, crawlID)
}

func (frontier *Frontier) ResumeQueueOf(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	state, err := frontier.queueStates.stateOf(tx, crawlID)
	if err != nil || state != queuePaused {
		return err
	}
	if err := frontier.queueStates.remove(tx, crawlID); err != nil {
		return err
	}
	return frontier.leases.resume(tx, crawlID)
}

func (frontier *Frontier) DrainQueueOf(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	state, err := frontier.queueStates.stateOf(tx, crawlID)
	if err != nil || state == queueDraining {
		return err
	}
	if err := frontier.leases.stop(tx, crawlID); err != nil {
		return err
	}
	return frontier.startDrainOf(tx, crawlID)
}

func (frontier *Frontier) startDrainOf(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	pending, err := frontier.ledger.hasPending(tx, crawlID)
	if err != nil {
		return err
	}
	if !pending {
		return frontier.queueStates.remove(tx, crawlID)
	}
	return frontier.queueStates.put(tx, crawlID, queueDraining)
}

func (frontier *Frontier) ForgetQueueOf(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if err := frontier.ledger.forgetCrawl(tx, crawlID); err != nil {
		return err
	}
	if err := frontier.leases.stop(tx, crawlID); err != nil {
		return err
	}
	return frontier.queueStates.remove(tx, crawlID)
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

func (frontier *Frontier) DrainQueues(ctx context.Context) error {
	for {
		if frontier.drainNextBatch(ctx) {
			continue
		}
		if err := frontier.clock.Sleep(ctx, noDrainingQueuePause); err != nil {
			return nil //nolint:nilerr // a clock only stops sleeping early when the drain is to end
		}
	}
}

func (frontier *Frontier) drainNextBatch(ctx context.Context) bool {
	crawlID, draining, err := frontier.pickedDrainingQueue(ctx)
	if err != nil {
		frontier.observer.FrontierDrainFailed(ctx, err)
		return false
	}
	if !draining {
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

func (frontier *Frontier) pickedDrainingQueue(
	ctx context.Context,
) (crawlid.CrawlID, bool, error) {
	var draining []crawlid.CrawlID
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		draining, err = frontier.queueStates.firstDrainingQueues(tx, drainingQueuesPickedAmong)
		return err
	}); err != nil {
		return crawlid.CrawlID{}, false, fmt.Errorf("look for a queue to drain: %w", err)
	}
	if len(draining) == 0 {
		return crawlid.CrawlID{}, false, nil
	}
	return draining[rand.IntN(len(draining))], true, nil //nolint:gosec // spreads processes, no secret
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
	amountsDrainedByHost, amountDrained, err := frontier.ledger.drainFrom(
		tx, crawlID, start, pageVisitsDrainedAtOnce,
	)
	if err != nil {
		return err
	}
	for _, host := range slices.Sorted(maps.Keys(amountsDrainedByHost)) {
		if err := frontier.drained.PageVisitsDrained(
			tx, crawlID, host, amountsDrainedByHost[host],
		); err != nil {
			return err
		}
	}
	if amountDrained == pageVisitsDrainedAtOnce {
		return nil
	}
	return frontier.queueStates.remove(tx, crawlID)
}

func (frontier *Frontier) AmountPending(ctx context.Context) (int64, bool) {
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
