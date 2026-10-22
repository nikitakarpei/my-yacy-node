// Package frontier keeps every page visit a crawl still owes, in a queue of its crawl, each
// due at a time, and a mark of every visit it settled. A worker leases the earliest due
// visit of a crawl neither held nor closed, and the visit comes back when the lease ends
// before the worker settles it. A visit owed or settled before is not added again, and a
// closed or cancelled crawl takes none. A cancelled crawl's owed visits are dropped in
// batches and reported as cancelled. The transaction that leaves a crawl owing nothing
// reports it. A pause holds every page visit.
package frontier

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const (
	frontierName            = "YACY_CRAWL_FRONTIER"
	frontierIndexesVersion  = "1"
	noDueVisitPause         = 500 * time.Millisecond
	noCancelledCrawlPause   = 500 * time.Millisecond
	pageVisitsDrainedAtOnce = 100
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type FrontierObserver interface {
	FrontierLeaseFailed(ctx context.Context, err error)
	FrontierDrainFailed(ctx context.Context, err error)
	FrontierCountFailed(ctx context.Context, err error)
	FrontierAcknowledgeFailed(ctx context.Context, identity string, err error)
	FrontierReturnFailed(ctx context.Context, identity string, delay time.Duration, err error)
	FrontierLeaseOutdated(ctx context.Context, identity string)
	FrontierCrawlNotSettled(ctx context.Context, crawlID crawlid.CrawlID, err error)
}

type CrawlSettlement interface {
	CrawlOwesNothing(tx *vault.Txn, crawlID crawlid.CrawlID) error
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

func (frontier *Frontier) Owe(tx *vault.Txn, pageVisits []pendingpagevisit.PendingPageVisit) error {
	now := frontier.clock.Now()
	for crawlID, pageVisitsOfCrawl := range pageVisitsByCrawl(pageVisits) {
		if err := frontier.oweOfCrawl(tx, crawlID, pageVisitsOfCrawl, now); err != nil {
			return fmt.Errorf("owe page visits of crawl %s: %w", crawlID, err)
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

func (frontier *Frontier) oweOfCrawl(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	pageVisits []pendingpagevisit.PendingPageVisit,
	now time.Time,
) error {
	mark, err := frontier.crawlMarks.markOf(tx, crawlID)
	if err != nil || !mark.takesPageVisits() {
		return err
	}
	owedAny := false
	for _, pageVisit := range pageVisits {
		owedNow, err := frontier.ledger.owe(tx, pageVisit, now)
		if err != nil {
			return err
		}
		owedAny = owedAny || owedNow
	}
	if !owedAny || mark == markHeld {
		return nil
	}
	return frontier.heads.moveEarlierTo(tx, crawlID, now.Unix())
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
	return frontier.restoreHeadOf(tx, crawlID)
}

func (frontier *Frontier) restoreHeadOf(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	first, owed, err := frontier.ledger.firstDueOf(tx, crawlID)
	if err != nil || !owed {
		return err
	}
	return frontier.heads.moveTo(tx, crawlID, first.due)
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
	owes, err := frontier.ledger.owesAny(tx, crawlID)
	if err != nil {
		return err
	}
	cancellingMark := markClosed
	if owes {
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

func (frontier *Frontier) AmountOwedBy(tx *vault.Txn, crawlID crawlid.CrawlID) (int, error) {
	return frontier.ledger.amountOwedBy(tx, crawlID)
}

func (frontier *Frontier) AmountsOwedByHostsOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	fn func(host string, amount int),
) error {
	return frontier.ledger.amountsOwedByHostsOf(tx, crawlID, fn)
}

func (frontier *Frontier) Next(ctx context.Context) (pullintake.PendingMessage, error) {
	for {
		message, leased, retry := frontier.leaseNextPageVisit(ctx)
		if leased {
			return message, nil
		}
		if retry {
			continue
		}
		if err := frontier.clock.Sleep(ctx, noDueVisitPause); err != nil {
			return nil, err
		}
	}
}

func (frontier *Frontier) leaseNextPageVisit(
	ctx context.Context,
) (message pendingPageVisitMessage, leased bool, retry bool) {
	now := frontier.clock.Now()
	head, due, err := frontier.firstHeadDue(ctx, now)
	if err != nil {
		frontier.observer.FrontierLeaseFailed(ctx, err)
		return pendingPageVisitMessage{}, false, false
	}
	if !due {
		return pendingPageVisitMessage{}, false, false
	}
	if err := frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		message, leased, err = frontier.leaseHeadOf(tx, head.crawlID, now)
		return err
	}); err != nil {
		frontier.observer.FrontierLeaseFailed(ctx, fmt.Errorf("lease a due page visit: %w", err))
		return pendingPageVisitMessage{}, false, false
	}
	return message, leased, !leased
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
) (pendingPageVisitMessage, bool, error) {
	paused, err := frontier.pause.isOn(tx)
	if err != nil || paused {
		return pendingPageVisitMessage{}, false, err
	}
	headed, err := frontier.heads.has(tx, crawlID)
	if err != nil || !headed {
		return pendingPageVisitMessage{}, false, err
	}
	first, owed, err := frontier.ledger.firstDueOf(tx, crawlID)
	if err != nil {
		return pendingPageVisitMessage{}, false, err
	}
	if !owed {
		return pendingPageVisitMessage{}, false, frontier.heads.remove(tx, crawlID)
	}
	if first.due > now.Unix() {
		return pendingPageVisitMessage{}, false, frontier.heads.moveTo(tx, crawlID, first.due)
	}
	message, err := frontier.leaseIn(tx, first.key, now)
	if err != nil {
		return pendingPageVisitMessage{}, false, err
	}
	return message, true, frontier.restoreHeadOf(tx, crawlID)
}

func (frontier *Frontier) leaseIn(
	tx *vault.Txn,
	key pageVisitKey,
	now time.Time,
) (pendingPageVisitMessage, error) {
	visit, err := frontier.ledger.lease(tx, key, secondsAtOrAfter(now.Add(frontier.lease)))
	if err != nil {
		return pendingPageVisitMessage{}, err
	}
	return pendingPageVisitMessage{
		frontier: frontier,
		key:      key,
		lease:    visit.due,
		identity: frontierName + "/" + hex.EncodeToString(visit.messageID[:]),
		arrival:  time.Unix(visit.arrival, 0),
		body:     visit.body,
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
	crawlID, cancelling, err := frontier.firstCancellingCrawl(ctx)
	if err != nil {
		frontier.observer.FrontierDrainFailed(ctx, err)
		return false
	}
	if !cancelling {
		return false
	}
	if err := frontier.settled(ctx, crawlID, func(
		tx *vault.Txn,
		settlement CrawlSettlement,
	) error {
		return frontier.drainBatchOf(tx, crawlID, settlement)
	}); err != nil {
		frontier.observer.FrontierDrainFailed(ctx, fmt.Errorf("drain crawl %s: %w", crawlID, err))
		return false
	}
	return true
}

func (frontier *Frontier) firstCancellingCrawl(
	ctx context.Context,
) (crawlid.CrawlID, bool, error) {
	var first crawlid.CrawlID
	var found bool
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		first, found, err = frontier.crawlMarks.firstCancellingCrawl(tx)
		return err
	}); err != nil {
		return crawlid.CrawlID{}, false, fmt.Errorf("look for a crawl to drain: %w", err)
	}
	return first, found, nil
}

func (frontier *Frontier) drainBatchOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	settlement CrawlSettlement,
) error {
	drainedOfHosts, amountDrained, err := frontier.ledger.drainFirstOf(
		tx, crawlID, pageVisitsDrainedAtOnce,
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
	return frontier.closeOwingNothing(tx, crawlID, settlement)
}

func (frontier *Frontier) closeOwingNothing(
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
	if err := settlement.CrawlOwesNothing(tx, crawlID); err != nil {
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

func (unsettledCrawls) CrawlOwesNothing(*vault.Txn, crawlid.CrawlID) error {
	return nil
}

func (frontier *Frontier) AmountOfPendingPageVisits(ctx context.Context) (int64, bool) {
	var amount int
	if err := frontier.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		amount, err = frontier.ledger.amountOwed(tx)
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

func (frontier *Frontier) acknowledge(ctx context.Context, message pendingPageVisitMessage) {
	leased := false
	if err := frontier.settled(ctx, message.key.crawlID, func(
		tx *vault.Txn,
		settlement CrawlSettlement,
	) error {
		var err error
		leased, err = frontier.settleIn(tx, message, settlement)
		return err
	}); err != nil {
		frontier.observer.FrontierAcknowledgeFailed(ctx, message.identity, err)
		return
	}
	if !leased {
		frontier.observer.FrontierLeaseOutdated(ctx, message.identity)
	}
}

func (frontier *Frontier) settleIn(
	tx *vault.Txn,
	message pendingPageVisitMessage,
	settlement CrawlSettlement,
) (bool, error) {
	settled, err := frontier.ledger.settleLeased(tx, message.key, message.lease)
	if err != nil || !settled {
		return false, err
	}
	owes, err := frontier.ledger.owesAny(tx, message.key.crawlID)
	if err != nil || owes {
		return true, err
	}
	return true, frontier.closeOwingNothing(tx, message.key.crawlID, settlement)
}

func (frontier *Frontier) returnAfter(
	ctx context.Context,
	message pendingPageVisitMessage,
	delay time.Duration,
) {
	due := secondsAtOrAfter(frontier.clock.Now().Add(delay))
	leased := false
	if err := frontier.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		leased, err = frontier.ledger.returnLeased(tx, message.key, message.lease, due)
		if err != nil || !leased {
			return err
		}
		return frontier.moveHeadEarlierOf(tx, message.key.crawlID, due)
	}); err != nil {
		frontier.observer.FrontierReturnFailed(ctx, message.identity, delay, err)
		return
	}
	if !leased {
		frontier.observer.FrontierLeaseOutdated(ctx, message.identity)
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
	return frontier.heads.moveEarlierTo(tx, crawlID, due)
}
