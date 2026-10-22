// Package frontier keeps every pending page visit of a crawl, in a queue of its crawl, each
// due at a time, and a mark of every visit it settled. A worker leases the earliest due
// visit of a crawl neither held nor closed, and the visit comes back when the lease ends
// before the worker settles it. A visit pending or settled before is not added again, and a
// closed or cancelled crawl takes none. A cancelled crawl's pending visits are dropped in
// batches and reported as cancelled. A running crawl with nothing pending is reported when
// its head falls due, a held or cancelled one by the transaction that leaves it so. A pause
// holds every page visit.
package frontier

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
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
	FrontierSettleFailed(
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
	FrontierCrawlNotSettled(ctx context.Context, crawlID crawlid.CrawlID, err error)
}

type CrawlSettlement interface {
	LastPageVisitEnded(tx *vault.Txn, crawlID crawlid.CrawlID) error
}

type CancelledPageVisits interface {
	PageVisitsCancelled(tx *vault.Txn, crawlID crawlid.CrawlID, host string, amount uint) error
}

type Frontier struct {
	storage    *vault.Vault
	settlement CrawlSettlement
	cancelled  CancelledPageVisits
	ledger     pageVisitLedger
	heads      crawlHeads
	crawlMarks crawlMarks
	pause      pageVisitPause
	lease      time.Duration
	clock      Clock
	observer   FrontierObserver
}

//nolint:revive // a frontier names the vault, the ports it reports to, its lease, clock and observer
func Open(
	ctx context.Context,
	storage *vault.Vault,
	settlement CrawlSettlement,
	cancelled CancelledPageVisits,
	lease time.Duration,
	clock Clock,
	observer FrontierObserver,
) (*Frontier, error) {
	frontier := &Frontier{
		storage:    storage,
		settlement: settlement,
		cancelled:  cancelled,
		lease:      lease,
		clock:      clock,
		observer:   observer,
	}
	var err error
	if frontier.ledger, err = registerPageVisitLedger(ctx, storage); err != nil {
		return nil, err
	}
	if frontier.heads, err = registerCrawlHeads(ctx, storage); err != nil {
		return nil, err
	}
	if frontier.crawlMarks, err = registerCrawlMarks(ctx, storage); err != nil {
		return nil, err
	}
	if frontier.pause, err = registerPageVisitPause(storage); err != nil {
		return nil, err
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
	return frontier.heads.moveEarlierTo(tx, crawlID, duePosition{due: now.Unix()})
}

func (frontier *Frontier) Hold(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || mark != unmarked {
		return err
	}
	if err := frontier.crawlMarks.mark(tx, crawlID, markHeld); err != nil {
		return err
	}
	return frontier.heads.remove(tx, crawlID)
}

func (frontier *Frontier) Release(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || mark != markHeld {
		return err
	}
	if err := frontier.crawlMarks.unmark(tx, crawlID); err != nil {
		return err
	}
	return frontier.restoreHeadFrom(tx, crawlID, earliestPosition)
}

func (frontier *Frontier) restoreHeadFrom(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	position duePosition,
) error {
	first, pending, err := frontier.ledger.firstDueFrom(tx, crawlID, position)
	if err != nil {
		return err
	}
	if !pending {
		return frontier.heads.moveTo(tx, crawlID, earliestPosition)
	}
	return frontier.heads.moveTo(tx, crawlID, first.position())
}

func (frontier *Frontier) Close(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || !mark.takesPageVisits() {
		return err
	}
	if err := frontier.crawlMarks.mark(tx, crawlID, markClosed); err != nil {
		return err
	}
	return frontier.heads.remove(tx, crawlID)
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
	return frontier.heads.remove(tx, crawlID)
}

func (frontier *Frontier) ForgetCrawl(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	if err := frontier.ledger.forgetCrawl(tx, crawlID); err != nil {
		return err
	}
	if err := frontier.heads.remove(tx, crawlID); err != nil {
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
	for {
		leasedPageVisit, leased, retry := frontier.leaseNextPageVisit(ctx)
		if leased {
			return leasedPageVisit, nil
		}
		if retry {
			continue
		}
		if err := frontier.clock.Sleep(ctx, noDueVisitPause); err != nil {
			return pendingpagevisit.LeasedPageVisit{}, err
		}
	}
}

func (frontier *Frontier) leaseNextPageVisit(
	ctx context.Context,
) (leasedPageVisit pendingpagevisit.LeasedPageVisit, leased bool, retry bool) {
	now := frontier.clock.Now()
	head, due, err := frontier.firstHeadDue(ctx, now)
	if err != nil {
		frontier.observer.FrontierLeaseFailed(ctx, err)
		return pendingpagevisit.LeasedPageVisit{}, false, false
	}
	if !due {
		return pendingpagevisit.LeasedPageVisit{}, false, false
	}
	if err := frontier.settled(ctx, head.crawlID, func(
		tx *vault.Txn,
		settlement CrawlSettlement,
	) error {
		var err error
		leasedPageVisit, leased, err = frontier.leaseHeadOf(tx, head.crawlID, now, settlement)
		return err
	}); err != nil {
		frontier.observer.FrontierLeaseFailed(ctx, fmt.Errorf("lease a due page visit: %w", err))
		return pendingpagevisit.LeasedPageVisit{}, false, false
	}
	return leasedPageVisit, leased, !leased
}

func (frontier *Frontier) firstHeadDue(
	ctx context.Context,
	now time.Time,
) (crawlHead, bool, error) {
	var head crawlHead
	var due bool
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		paused, err := frontier.pause.isOn(tx)
		if err != nil || paused {
			return err
		}
		head, due, err = frontier.heads.firstDueBy(tx, now)
		return err
	}); err != nil {
		return crawlHead{}, false, fmt.Errorf("look for a due page visit: %w", err)
	}
	return head, due, nil
}

func (frontier *Frontier) leaseHeadOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	now time.Time,
	settlement CrawlSettlement,
) (pendingpagevisit.LeasedPageVisit, bool, error) {
	paused, err := frontier.pause.isOn(tx)
	if err != nil || paused {
		return pendingpagevisit.LeasedPageVisit{}, false, err
	}
	headPosition, headed, err := frontier.heads.positionOf(tx, crawlID)
	if err != nil || !headed {
		return pendingpagevisit.LeasedPageVisit{}, false, err
	}
	first, pending, err := frontier.ledger.firstDueFrom(tx, crawlID, headPosition)
	if err != nil {
		return pendingpagevisit.LeasedPageVisit{}, false, err
	}
	if !pending {
		if err := frontier.heads.remove(tx, crawlID); err != nil {
			return pendingpagevisit.LeasedPageVisit{}, false, err
		}
		return pendingpagevisit.LeasedPageVisit{}, false, frontier.closeWithNothingPending(
			tx, crawlID, settlement,
		)
	}
	if first.due > now.Unix() {
		return pendingpagevisit.LeasedPageVisit{}, false, frontier.heads.moveTo(
			tx,
			crawlID,
			first.position(),
		)
	}
	leasedPageVisit, err := frontier.leaseIn(tx, first.key, now)
	if err != nil {
		return pendingpagevisit.LeasedPageVisit{}, false, err
	}
	return leasedPageVisit, true, frontier.restoreHeadFrom(tx, crawlID, first.position())
}

func (frontier *Frontier) leaseIn(
	tx *vault.Txn,
	key pageVisitKey,
	now time.Time,
) (pendingpagevisit.LeasedPageVisit, error) {
	visit, err := frontier.ledger.lease(tx, key, secondsAtOrAfter(now.Add(frontier.lease)))
	if err != nil {
		return pendingpagevisit.LeasedPageVisit{}, err
	}
	return pendingpagevisit.LeasedPageVisit{
		PageVisit: pendingpagevisit.PendingPageVisit{
			CrawlID: key.crawlID,
			URL:     visit.url,
			Depth:   visit.depth,
		},
		LeasedUntil: time.Unix(visit.due, 0),
	}, nil
}

func secondsAtOrAfter(at time.Time) int64 {
	seconds := at.Unix()
	if at.Nanosecond() > 0 {
		seconds++
	}
	return seconds
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
	if err := frontier.settled(ctx, crawlID, func(
		tx *vault.Txn,
		settlement CrawlSettlement,
	) error {
		return frontier.drainBatchOf(tx, crawlID, start, settlement)
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
	settlement CrawlSettlement,
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
	return frontier.closeWithNothingPending(tx, crawlID, settlement)
}

func (frontier *Frontier) closeWithNothingPending(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	settlement CrawlSettlement,
) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil {
		return err
	}
	if mark == markCancelling {
		if err := frontier.crawlMarks.mark(tx, crawlID, markClosed); err != nil {
			return err
		}
	}
	if err := settlement.LastPageVisitEnded(tx, crawlID); err != nil {
		return crawlSettlementError{cause: err}
	}
	return nil
}

type crawlSettlementError struct {
	cause error
}

func (e crawlSettlementError) Error() string {
	return e.cause.Error()
}

func (e crawlSettlementError) Unwrap() error {
	return e.cause
}

func (frontier *Frontier) settled(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	write func(tx *vault.Txn, settlement CrawlSettlement) error,
) error {
	err := frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		return write(tx, frontier.settlement)
	})
	notSettled, failed := errors.AsType[crawlSettlementError](err)
	if !failed {
		return err
	}
	frontier.observer.FrontierCrawlNotSettled(ctx, crawlID, notSettled.cause)
	return frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		return write(tx, unsettledCrawls{})
	})
}

type unsettledCrawls struct{}

func (unsettledCrawls) LastPageVisitEnded(*vault.Txn, crawlid.CrawlID) error {
	return nil
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

func (frontier *Frontier) Settle(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
) {
	pageVisit := leasedPageVisit.PageVisit
	leased := false
	if err := frontier.settled(ctx, pageVisit.CrawlID, func(
		tx *vault.Txn,
		settlement CrawlSettlement,
	) error {
		var err error
		leased, err = frontier.settleIn(tx, leasedPageVisit, settlement)
		return err
	}); err != nil {
		frontier.observer.FrontierSettleFailed(ctx, pageVisit, err)
		return
	}
	if !leased {
		frontier.observer.FrontierLeaseOutdated(ctx, pageVisit)
	}
}

func (frontier *Frontier) settleIn(
	tx *vault.Txn,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	settlement CrawlSettlement,
) (bool, error) {
	crawlID := leasedPageVisit.PageVisit.CrawlID
	settled, err := frontier.ledger.settleLeased(
		tx,
		pageVisitKeyOf(leasedPageVisit.PageVisit),
		leasedPageVisit.LeasedUntil.Unix(),
	)
	if err != nil || !settled {
		return false, err
	}
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || mark == unmarked {
		return true, err
	}
	pending, err := frontier.ledger.hasPending(tx, crawlID)
	if err != nil || pending {
		return true, err
	}
	return true, frontier.closeWithNothingPending(tx, crawlID, settlement)
}

func (frontier *Frontier) Return(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	delay time.Duration,
) {
	pageVisit := leasedPageVisit.PageVisit
	due := secondsAtOrAfter(frontier.clock.Now().Add(delay))
	leased := false
	if err := frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		leased, err = frontier.ledger.returnLeased(
			tx,
			pageVisitKeyOf(pageVisit),
			leasedPageVisit.LeasedUntil.Unix(),
			due,
		)
		if err != nil || !leased {
			return err
		}
		return frontier.moveHeadEarlierOf(tx, pageVisit.CrawlID, due)
	}); err != nil {
		frontier.observer.FrontierReturnFailed(ctx, pageVisit, delay, err)
		return
	}
	if !leased {
		frontier.observer.FrontierLeaseOutdated(ctx, pageVisit)
	}
}

func (frontier *Frontier) moveHeadEarlierOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	due int64,
) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || mark != unmarked {
		return err
	}
	return frontier.heads.moveEarlierTo(tx, crawlID, duePosition{due: due})
}
