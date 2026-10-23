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

func (leases pageVisitLeases) leaseUpTo(
	ctx context.Context,
	amount int,
) ([]pendingpagevisit.LeasedPageVisit, error) {
	for {
		leasedPageVisits, retry := leases.leaseNextPageVisits(ctx, amount)
		if len(leasedPageVisits) > 0 {
			return leasedPageVisits, nil
		}
		if retry {
			continue
		}
		if err := leases.clock.Sleep(ctx, noDueVisitPause); err != nil {
			return nil, err
		}
	}
}

func (leases pageVisitLeases) leaseNextPageVisits(
	ctx context.Context,
	amount int,
) (leasedPageVisits []pendingpagevisit.LeasedPageVisit, retry bool) {
	now := leases.clock.Now()
	head, due, err := leases.firstHeadDue(ctx, now)
	if err != nil {
		leases.observer.FrontierLeaseFailed(ctx, err)
		return nil, false
	}
	if !due {
		return nil, false
	}
	if err := leases.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		leasedPageVisits, err = leases.leaseDueVisitsOf(tx, head.crawlID, now, amount)
		return err
	}); err != nil {
		leases.observer.FrontierLeaseFailed(ctx, fmt.Errorf("lease due page visits: %w", err))
		return nil, false
	}
	return leasedPageVisits, len(leasedPageVisits) == 0
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

func (leases pageVisitLeases) leaseDueVisitsOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	now time.Time,
	amount int,
) ([]pendingpagevisit.LeasedPageVisit, error) {
	paused, err := leases.pause.isOn(tx)
	if err != nil || paused {
		return nil, err
	}
	position, headed, err := leases.heads.positionOf(tx, crawlID)
	if err != nil || !headed {
		return nil, err
	}
	var leasedPageVisits []pendingpagevisit.LeasedPageVisit
	for len(leasedPageVisits) < amount {
		first, pending, err := leases.ledger.firstDueFrom(tx, crawlID, position)
		if err != nil {
			return nil, err
		}
		if !pending || first.due > now.Unix() {
			break
		}
		leasedPageVisit, err := leases.leaseVisitAt(tx, first.key, now)
		if err != nil {
			return nil, err
		}
		leasedPageVisits = append(leasedPageVisits, leasedPageVisit)
		position = first.position()
	}
	return leasedPageVisits, leases.alignHeadFrom(tx, crawlID, position)
}

func (leases pageVisitLeases) alignHeadFrom(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	position duePosition,
) error {
	first, pending, err := leases.ledger.firstDueFrom(tx, crawlID, position)
	if err != nil {
		return err
	}
	if !pending {
		return leases.heads.remove(tx, crawlID)
	}
	return leases.heads.moveTo(tx, crawlID, first.position())
}

func (leases pageVisitLeases) leaseVisitAt(
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

func (leases pageVisitLeases) end(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
) {
	pageVisit := leasedPageVisit.PageVisit
	ended := false
	if err := leases.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		ended, err = leases.ledger.endLeased(
			tx,
			pageVisitKeyOf(pageVisit),
			leasedPageVisit.LeasedUntil.Unix(),
		)
		return err
	}); err != nil {
		leases.observer.FrontierEndFailed(ctx, pageVisit, err)
		return
	}
	if !ended {
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
	returned := false
	if err := leases.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		returned, err = leases.ledger.returnLeased(
			tx,
			pageVisitKeyOf(pageVisit),
			leasedPageVisit.LeasedUntil.Unix(),
			due,
		)
		if err != nil || !returned {
			return err
		}
		return leases.heads.moveEarlierIfPresent(tx, pageVisit.CrawlID, duePosition{due: due})
	}); err != nil {
		leases.observer.FrontierReturnFailed(ctx, pageVisit, delay, err)
		return
	}
	if !returned {
		leases.observer.FrontierLeaseOutdated(ctx, pageVisit)
	}
}

func (leases pageVisitLeases) startFrom(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	due time.Time,
) error {
	return leases.heads.moveEarlierTo(tx, crawlID, duePosition{due: due.Unix()})
}

func (leases pageVisitLeases) stop(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return leases.heads.remove(tx, crawlID)
}

func (leases pageVisitLeases) resume(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	return leases.alignHeadFrom(tx, crawlID, earliestPosition)
}
