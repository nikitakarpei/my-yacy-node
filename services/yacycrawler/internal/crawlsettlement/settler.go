// Package crawlsettlement settles every unfinished crawl that has no pending page visit
// left: a running or paused crawl succeeds, a cancelling one is cancelled. It looks at
// the unfinished crawls again after a short pause, so a crawl settles shortly after its
// last page visit ends, whichever write ended it, and one it could not settle is tried again.
package crawlsettlement

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const settlementPause = time.Second

type Crawls interface {
	UnfinishedCrawlIDs(tx *vault.Txn) ([]crawlid.CrawlID, error)
	Settle(tx *vault.Txn, crawlID crawlid.CrawlID) error
}

type PendingPageVisits interface {
	AmountPendingOf(tx *vault.Txn, crawlID crawlid.CrawlID) (int, error)
}

type Clock interface {
	Sleep(ctx context.Context, d time.Duration) error
}

type SettlerObserver interface {
	UnfinishedCrawlsNotRead(ctx context.Context, err error)
	CrawlNotSettled(ctx context.Context, crawlID crawlid.CrawlID, err error)
}

type Settler struct {
	storage           *vault.Vault
	crawls            Crawls
	pendingPageVisits PendingPageVisits
	clock             Clock
	observer          SettlerObserver
}

func NewSettler(
	storage *vault.Vault,
	crawls Crawls,
	pendingPageVisits PendingPageVisits,
	clock Clock,
	observer SettlerObserver,
) *Settler {
	return &Settler{
		storage:           storage,
		crawls:            crawls,
		pendingPageVisits: pendingPageVisits,
		clock:             clock,
		observer:          observer,
	}
}

func (settler *Settler) Run(ctx context.Context) error {
	for {
		settler.settleEmptyCrawls(ctx)
		if err := settler.clock.Sleep(ctx, settlementPause); err != nil {
			return nil //nolint:nilerr // a clock only stops sleeping early when settling is to end
		}
	}
}

func (settler *Settler) settleEmptyCrawls(ctx context.Context) {
	crawlIDs, err := settler.unfinishedCrawlIDs(ctx)
	if err != nil {
		settler.observer.UnfinishedCrawlsNotRead(ctx, err)
		return
	}
	for _, crawlID := range crawlIDs {
		if err := settler.settleIfEmpty(ctx, crawlID); err != nil {
			settler.observer.CrawlNotSettled(ctx, crawlID, err)
		}
	}
}

func (settler *Settler) unfinishedCrawlIDs(ctx context.Context) ([]crawlid.CrawlID, error) {
	var crawlIDs []crawlid.CrawlID
	if err := settler.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		crawlIDs, err = settler.crawls.UnfinishedCrawlIDs(tx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("read the unfinished crawls: %w", err)
	}
	return crawlIDs, nil
}

func (settler *Settler) settleIfEmpty(ctx context.Context, crawlID crawlid.CrawlID) error {
	empty, err := settler.isEmpty(ctx, crawlID)
	if err != nil || !empty {
		return err
	}
	if err := settler.storage.Update(ctx, func(tx *vault.Txn) error {
		return settler.settleInIfEmpty(tx, crawlID)
	}); err != nil {
		return fmt.Errorf("settle the crawl: %w", err)
	}
	return nil
}

func (settler *Settler) isEmpty(ctx context.Context, crawlID crawlid.CrawlID) (bool, error) {
	var empty bool
	if err := settler.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		empty, err = settler.hasNothingPending(tx, crawlID)
		return err
	}); err != nil {
		return false, fmt.Errorf("count the pending page visits of the crawl: %w", err)
	}
	return empty, nil
}

func (settler *Settler) hasNothingPending(tx *vault.Txn, crawlID crawlid.CrawlID) (bool, error) {
	amountPending, err := settler.pendingPageVisits.AmountPendingOf(tx, crawlID)
	return amountPending == 0, err
}

func (settler *Settler) settleInIfEmpty(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	empty, err := settler.hasNothingPending(tx, crawlID)
	if err != nil || !empty {
		return err
	}
	return settler.crawls.Settle(tx, crawlID)
}
