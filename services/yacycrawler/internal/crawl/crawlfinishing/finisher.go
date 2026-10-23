// Package crawlfinishing finishes every unfinished crawl that has no pending page visit
// left: a running or paused crawl succeeds, a cancelling one is cancelled. It looks at
// the unfinished crawls again after a short pause, so a crawl finishes shortly after its
// last page visit ends, whichever write ended it, and one it could not finish is tried again.
package crawlfinishing

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const finishingPause = time.Second

type Crawls interface {
	UnfinishedCrawlIDs(tx *vault.Txn) ([]crawlid.CrawlID, error)
	Finish(tx *vault.Txn, crawlID crawlid.CrawlID) error
}

type PendingPageVisits interface {
	AmountPendingOf(tx *vault.Txn, crawlID crawlid.CrawlID) (int, error)
}

type Clock interface {
	Sleep(ctx context.Context, d time.Duration) error
}

type FinisherObserver interface {
	UnfinishedCrawlsNotRead(ctx context.Context, err error)
	CrawlNotFinished(ctx context.Context, crawlID crawlid.CrawlID, err error)
}

type Finisher struct {
	storage           *vault.Vault
	crawls            Crawls
	pendingPageVisits PendingPageVisits
	clock             Clock
	observer          FinisherObserver
}

func NewFinisher(
	storage *vault.Vault,
	crawls Crawls,
	pendingPageVisits PendingPageVisits,
	clock Clock,
	observer FinisherObserver,
) *Finisher {
	return &Finisher{
		storage:           storage,
		crawls:            crawls,
		pendingPageVisits: pendingPageVisits,
		clock:             clock,
		observer:          observer,
	}
}

func (finisher *Finisher) Run(ctx context.Context) error {
	for {
		finisher.finishEmptyCrawls(ctx)
		if err := finisher.clock.Sleep(ctx, finishingPause); err != nil {
			return nil //nolint:nilerr // a clock only stops sleeping early when finishing is to stop
		}
	}
}

func (finisher *Finisher) finishEmptyCrawls(ctx context.Context) {
	crawlIDs, err := finisher.unfinishedCrawlIDs(ctx)
	if err != nil {
		finisher.observer.UnfinishedCrawlsNotRead(ctx, err)
		return
	}
	for _, crawlID := range crawlIDs {
		if err := finisher.finishIfEmpty(ctx, crawlID); err != nil {
			finisher.observer.CrawlNotFinished(ctx, crawlID, err)
		}
	}
}

func (finisher *Finisher) unfinishedCrawlIDs(ctx context.Context) ([]crawlid.CrawlID, error) {
	var crawlIDs []crawlid.CrawlID
	if err := finisher.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		crawlIDs, err = finisher.crawls.UnfinishedCrawlIDs(tx)
		return err
	}); err != nil {
		return nil, fmt.Errorf("read the unfinished crawls: %w", err)
	}
	return crawlIDs, nil
}

func (finisher *Finisher) finishIfEmpty(ctx context.Context, crawlID crawlid.CrawlID) error {
	empty, err := finisher.isEmpty(ctx, crawlID)
	if err != nil || !empty {
		return err
	}
	if err := finisher.storage.Update(ctx, func(tx *vault.Txn) error {
		return finisher.finishInIfEmpty(tx, crawlID)
	}); err != nil {
		return fmt.Errorf("finish the crawl: %w", err)
	}
	return nil
}

func (finisher *Finisher) isEmpty(ctx context.Context, crawlID crawlid.CrawlID) (bool, error) {
	var empty bool
	if err := finisher.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		empty, err = finisher.hasNothingPending(tx, crawlID)
		return err
	}); err != nil {
		return false, fmt.Errorf("count the pending page visits of the crawl: %w", err)
	}
	return empty, nil
}

func (finisher *Finisher) hasNothingPending(tx *vault.Txn, crawlID crawlid.CrawlID) (bool, error) {
	amountPending, err := finisher.pendingPageVisits.AmountPendingOf(tx, crawlID)
	return amountPending == 0, err
}

func (finisher *Finisher) finishInIfEmpty(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	empty, err := finisher.hasNothingPending(tx, crawlID)
	if err != nil || !empty {
		return err
	}
	return finisher.crawls.Finish(tx, crawlID)
}
