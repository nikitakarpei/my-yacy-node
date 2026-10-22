// Package crawls keeps every crawl the crawler runs: the accepted order it runs for, its
// state, and when it started and finished. The first order of an ID starts its crawl; an
// order sent again finds that crawl while it is kept. Each state change is reported in its
// transaction. A finished crawl is kept for the crawl retention after it finishes, then
// forgotten, and its forgetting is reported in the same transaction.
package crawls

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

type Clock interface {
	Now() time.Time
}

type CrawlStateChanges interface {
	CrawlPaused(tx *vault.Txn, crawlID crawlid.CrawlID) error
	CrawlResumed(tx *vault.Txn, crawlID crawlid.CrawlID) error
	CrawlCancelling(tx *vault.Txn, crawlID crawlid.CrawlID) error
	CrawlFinished(tx *vault.Txn, crawlID crawlid.CrawlID) error
}

type ForgottenCrawls interface {
	CrawlForgotten(tx *vault.Txn, crawlID crawlid.CrawlID) error
}

type Crawl struct {
	Order      yacycrawlcontract.CrawlOrder
	State      yacycrawlcontract.CrawlState
	StartedAt  time.Time
	FinishedAt time.Time
}

type Crawls struct {
	unfinished        *vault.Collection[crawlid.CrawlID, Crawl]
	finished          *vault.ExpiringCollection[crawlid.CrawlID, Crawl]
	unfinishedIndexes crawlIndexes
	finishedIndexes   crawlIndexes
	amountsInState    *vault.CountIndex[crawlWithID, yacycrawlcontract.CrawlState]
	changes           CrawlStateChanges
	clock             Clock
}

//nolint:revive // the crawls name the vault, the retention, the ports they report to and a clock
func Open(
	ctx context.Context,
	storage *vault.Vault,
	retention time.Duration,
	changes CrawlStateChanges,
	forgotten ForgottenCrawls,
	clock Clock,
) (*Crawls, error) {
	c, err := registerCrawls(ctx, storage, retention, forgotten)
	if err != nil {
		return nil, err
	}
	c.changes, c.clock = changes, clock
	return c, nil
}

func (c *Crawls) Start(
	tx *vault.Txn,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, bool, error) {
	crawlID, found, err := c.keptCrawlOfOrder(tx, order.OrderID())
	if err != nil || found {
		return crawlID, false, err
	}
	if crawlID, err = crawlid.New(); err != nil {
		return crawlid.CrawlID{}, false, err
	}
	if _, err := c.unfinished.Put(tx, crawlID, Crawl{
		Order:     order.CrawlOrder(),
		State:     yacycrawlcontract.CrawlStateRunning,
		StartedAt: c.clock.Now(),
	}); err != nil {
		return crawlid.CrawlID{}, false, fmt.Errorf(
			"start the crawl of order %s: %w",
			order.OrderID(),
			err,
		)
	}
	return crawlID, true, nil
}

func (c *Crawls) keptCrawlOfOrder(tx *vault.Txn, orderID string) (crawlid.CrawlID, bool, error) {
	unfinishedIDs, err := c.unfinishedIndexes.crawlIDsOfOrder(tx, orderID)
	if err != nil {
		return crawlid.CrawlID{}, false, err
	}
	finishedIDs, err := c.finishedIndexes.crawlIDsOfOrder(tx, orderID)
	if err != nil {
		return crawlid.CrawlID{}, false, err
	}
	for _, crawlID := range slices.Concat(unfinishedIDs, finishedIDs) {
		_, kept, err := c.CrawlOf(tx, crawlID)
		if err != nil || kept {
			return crawlID, kept, err
		}
	}
	return crawlid.CrawlID{}, false, nil
}

func (c *Crawls) CrawlOf(tx *vault.Txn, crawlID crawlid.CrawlID) (Crawl, bool, error) {
	crawl, found, err := c.unfinished.Get(tx, crawlID)
	if err == nil && !found {
		crawl, found, err = c.finished.Get(tx, crawlID)
	}
	if err != nil {
		return Crawl{}, false, fmt.Errorf("read the crawl %s: %w", crawlID, err)
	}
	return crawl, found, nil
}

func (c *Crawls) OrderOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (acceptedorder.AcceptedOrder, bool, error) {
	crawl, found, err := c.CrawlOf(tx, crawlID)
	if err != nil || !found {
		return acceptedorder.AcceptedOrder{}, false, err
	}
	order, err := acceptedorder.AcceptedOrderFrom(crawl.Order)
	if err != nil {
		return acceptedorder.AcceptedOrder{}, false, err
	}
	return order, true, nil
}

func (c *Crawls) Pause(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (yacycrawlcontract.CrawlState, bool, error) {
	return c.changedState(
		tx,
		crawlID,
		isRunning,
		yacycrawlcontract.CrawlStatePaused,
		c.changes.CrawlPaused,
	)
}

func (c *Crawls) Resume(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (yacycrawlcontract.CrawlState, bool, error) {
	return c.changedState(
		tx,
		crawlID,
		isPaused,
		yacycrawlcontract.CrawlStateRunning,
		c.changes.CrawlResumed,
	)
}

func (c *Crawls) Cancel(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (yacycrawlcontract.CrawlState, bool, error) {
	return c.changedState(
		tx, crawlID, isRunningOrPaused,
		yacycrawlcontract.CrawlStateCancelling, c.changes.CrawlCancelling,
	)
}

func isRunning(state yacycrawlcontract.CrawlState) bool {
	return state == yacycrawlcontract.CrawlStateRunning
}

func isPaused(state yacycrawlcontract.CrawlState) bool {
	return state == yacycrawlcontract.CrawlStatePaused
}

func isRunningOrPaused(state yacycrawlcontract.CrawlState) bool {
	return isRunning(state) || isPaused(state)
}

func (c *Crawls) changedState(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	changesFrom func(yacycrawlcontract.CrawlState) bool,
	changedState yacycrawlcontract.CrawlState,
	reportChange func(tx *vault.Txn, crawlID crawlid.CrawlID) error,
) (yacycrawlcontract.CrawlState, bool, error) {
	crawl, found, err := c.CrawlOf(tx, crawlID)
	if err != nil || !found {
		return "", false, err
	}
	if !changesFrom(crawl.State) {
		return crawl.State, true, nil
	}
	crawl.State = changedState
	if _, err := c.unfinished.Put(tx, crawlID, crawl); err != nil {
		return "", false, fmt.Errorf("change the state of the crawl %s: %w", crawlID, err)
	}
	return changedState, true, reportChange(tx, crawlID)
}

func (c *Crawls) Settle(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	crawl, found, err := c.unfinished.Get(tx, crawlID)
	if err != nil {
		return fmt.Errorf("read the crawl %s: %w", crawlID, err)
	}
	if !found {
		return nil
	}
	crawl.State, crawl.FinishedAt = settledStateOf(crawl.State), c.clock.Now()
	if _, err := c.unfinished.Delete(tx, crawlID); err != nil {
		return fmt.Errorf("settle the crawl %s: %w", crawlID, err)
	}
	if err := c.finished.Put(tx, crawlID, crawl); err != nil {
		return fmt.Errorf("settle the crawl %s: %w", crawlID, err)
	}
	return c.changes.CrawlFinished(tx, crawlID)
}

func settledStateOf(state yacycrawlcontract.CrawlState) yacycrawlcontract.CrawlState {
	if state == yacycrawlcontract.CrawlStateCancelling {
		return yacycrawlcontract.CrawlStateCancelled
	}
	return yacycrawlcontract.CrawlStateSucceeded
}

func (c *Crawls) AmountOfRunningCrawls(tx *vault.Txn) (int, error) {
	return c.amountsInState.Get(tx, yacycrawlcontract.CrawlStateRunning)
}

func (c *Crawls) CrawlIDsOlderThan(
	tx *vault.Txn,
	state yacycrawlcontract.CrawlState,
	border crawlid.CrawlID,
	limit int,
) ([]crawlid.CrawlID, error) {
	return c.indexesOf(state).crawlIDsOlderThan(tx, state, border, limit)
}

func (c *Crawls) CrawlIDsNewerThan(
	tx *vault.Txn,
	state yacycrawlcontract.CrawlState,
	border crawlid.CrawlID,
	limit int,
) ([]crawlid.CrawlID, error) {
	return c.indexesOf(state).crawlIDsNewerThan(tx, state, border, limit)
}

func (c *Crawls) indexesOf(state yacycrawlcontract.CrawlState) crawlIndexes {
	if state == yacycrawlcontract.CrawlStateSucceeded ||
		state == yacycrawlcontract.CrawlStateCancelled {
		return c.finishedIndexes
	}
	return c.unfinishedIndexes
}
