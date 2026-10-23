package frontier

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type pageVisitLeases struct {
	storage  *vault.Vault
	ledger   pageVisitLedger
	heads    crawlHeads
	pause    pageVisitPause
	lease    time.Duration
	clock    Clock
	observer FrontierObserver
}

func (leases pageVisitLeases) next(ctx context.Context) (pendingpagevisit.LeasedPageVisit, error) {
	for {
		leasedPageVisit, leased, retry := leases.leaseNextPageVisit(ctx)
		if leased {
			return leasedPageVisit, nil
		}
		if retry {
			continue
		}
		if err := leases.clock.Sleep(ctx, noDueVisitPause); err != nil {
			return pendingpagevisit.LeasedPageVisit{}, err
		}
	}
}

func (leases pageVisitLeases) leaseNextPageVisit(
	ctx context.Context,
) (leasedPageVisit pendingpagevisit.LeasedPageVisit, leased bool, retry bool) {
	now := leases.clock.Now()
	head, due, err := leases.firstHeadDue(ctx, now)
	if err != nil {
		leases.observer.FrontierLeaseFailed(ctx, err)
		return pendingpagevisit.LeasedPageVisit{}, false, false
	}
	if !due {
		return pendingpagevisit.LeasedPageVisit{}, false, false
	}
	if err := leases.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		leasedPageVisit, leased, err = leases.leaseHeadOf(tx, head.crawlID, now)
		return err
	}); err != nil {
		leases.observer.FrontierLeaseFailed(ctx, fmt.Errorf("lease a due page visit: %w", err))
		return pendingpagevisit.LeasedPageVisit{}, false, false
	}
	return leasedPageVisit, leased, !leased
}

func (leases pageVisitLeases) firstHeadDue(
	ctx context.Context,
	now time.Time,
) (crawlHead, bool, error) {
	var head crawlHead
	var due bool
	if err := leases.storage.View(ctx, func(tx *vault.Txn) error {
		paused, err := leases.pause.isOn(tx)
		if err != nil || paused {
			return err
		}
		head, due, err = leases.heads.firstDueBy(tx, now)
		return err
	}); err != nil {
		return crawlHead{}, false, fmt.Errorf("look for a due page visit: %w", err)
	}
	return head, due, nil
}

func (leases pageVisitLeases) leaseHeadOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	now time.Time,
) (pendingpagevisit.LeasedPageVisit, bool, error) {
	paused, err := leases.pause.isOn(tx)
	if err != nil || paused {
		return pendingpagevisit.LeasedPageVisit{}, false, err
	}
	headPosition, headed, err := leases.heads.positionOf(tx, crawlID)
	if err != nil || !headed {
		return pendingpagevisit.LeasedPageVisit{}, false, err
	}
	first, pending, err := leases.ledger.firstDueFrom(tx, crawlID, headPosition)
	if err != nil {
		return pendingpagevisit.LeasedPageVisit{}, false, err
	}
	if !pending {
		return pendingpagevisit.LeasedPageVisit{}, false, leases.heads.remove(tx, crawlID)
	}
	if first.due > now.Unix() {
		return pendingpagevisit.LeasedPageVisit{}, false, leases.heads.moveTo(
			tx,
			crawlID,
			first.position(),
		)
	}
	leasedPageVisit, err := leases.leaseIn(tx, first.key, now)
	if err != nil {
		return pendingpagevisit.LeasedPageVisit{}, false, err
	}
	return leasedPageVisit, true, leases.rescheduleFrom(tx, crawlID, first.position())
}

func (leases pageVisitLeases) leaseIn(
	tx *vault.Txn,
	key pageVisitKey,
	now time.Time,
) (pendingpagevisit.LeasedPageVisit, error) {
	visit, err := leases.ledger.lease(tx, key, secondsAtOrAfter(now.Add(leases.lease)))
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

func (leases pageVisitLeases) rescheduleFrom(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	position duePosition,
) error {
	first, pending, err := leases.ledger.firstDueFrom(tx, crawlID, position)
	if err != nil || !pending {
		return err
	}
	return leases.heads.moveTo(tx, crawlID, first.position())
}

func (leases pageVisitLeases) settle(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
) {
	pageVisit := leasedPageVisit.PageVisit
	leased := false
	if err := leases.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		leased, err = leases.ledger.settleLeased(
			tx,
			pageVisitKeyOf(pageVisit),
			leasedPageVisit.LeasedUntil.Unix(),
		)
		return err
	}); err != nil {
		leases.observer.FrontierSettleFailed(ctx, pageVisit, err)
		return
	}
	if !leased {
		leases.observer.FrontierLeaseOutdated(ctx, pageVisit)
	}
}

func (leases pageVisitLeases) returnAfter(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	delay time.Duration,
) {
	pageVisit := leasedPageVisit.PageVisit
	due := secondsAtOrAfter(leases.clock.Now().Add(delay))
	leased := false
	if err := leases.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		leased, err = leases.ledger.returnLeased(
			tx,
			pageVisitKeyOf(pageVisit),
			leasedPageVisit.LeasedUntil.Unix(),
			due,
		)
		if err != nil || !leased {
			return err
		}
		return leases.heads.moveEarlierIfPresent(tx, pageVisit.CrawlID, duePosition{due: due})
	}); err != nil {
		leases.observer.FrontierReturnFailed(ctx, pageVisit, delay, err)
		return
	}
	if !leased {
		leases.observer.FrontierLeaseOutdated(ctx, pageVisit)
	}
}

func (leases pageVisitLeases) scheduleNoLaterThan(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	due time.Time,
) error {
	return leases.heads.moveEarlierTo(tx, crawlID, duePosition{due: due.Unix()})
}

func (leases pageVisitLeases) unschedule(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return leases.heads.remove(tx, crawlID)
}

func (leases pageVisitLeases) reschedule(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return leases.rescheduleFrom(tx, crawlID, earliestPosition)
}
