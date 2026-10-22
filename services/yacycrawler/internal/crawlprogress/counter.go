package crawlprogress

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type CounterObserver interface {
	CompletedPageVisitNotCounted(
		ctx context.Context,
		crawlID crawlid.CrawlID,
		result yacycrawlcontract.PageVisitResult,
		err error,
	)
	DroppedPageVisitNotCounted(
		ctx context.Context,
		crawlID crawlid.CrawlID,
		reason yacycrawlcontract.PageVisitDropReason,
		err error,
	)
	UnadmittedLinksNotCounted(ctx context.Context, crawlID crawlid.CrawlID, err error)
}

type Counter struct {
	storage  *vault.Vault
	counts   *Counts
	observer CounterObserver
}

func NewCounter(storage *vault.Vault, counts *Counts, observer CounterObserver) *Counter {
	return &Counter{storage: storage, counts: counts, observer: observer}
}

func (counter *Counter) PendingPageVisitCompleted(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	result yacycrawlcontract.PageVisitResult,
) {
	crawlID := pendingPageVisit.CrawlID
	if err := counter.storage.Update(ctx, func(tx *vault.Txn) error {
		return counter.counts.CountCompleted(tx, crawlID, pendingPageVisit.URL.Hostname(), result)
	}); err != nil {
		counter.observer.CompletedPageVisitNotCounted(ctx, crawlID, result, err)
	}
}

func (counter *Counter) PendingPageVisitDropped(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	reason yacycrawlcontract.PageVisitDropReason,
) {
	crawlID := pendingPageVisit.CrawlID
	if err := counter.storage.Update(ctx, func(tx *vault.Txn) error {
		return counter.counts.CountDropped(tx, crawlID, pendingPageVisit.URL.Hostname(), reason, 1)
	}); err != nil {
		counter.observer.DroppedPageVisitNotCounted(ctx, crawlID, reason, err)
	}
}

func (counter *Counter) LinksUnadmitted(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	amountOfLinks int,
) {
	if err := counter.storage.Update(ctx, func(tx *vault.Txn) error {
		return counter.counts.CountUnadmittedLinks(tx, crawlID, uint(amountOfLinks))
	}); err != nil {
		counter.observer.UnadmittedLinksNotCounted(ctx, crawlID, err)
	}
}

func (*Counter) PendingPageVisitReturned(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	error,
) {
}

func (*Counter) PendingPageVisitDeferred(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	time.Duration,
) {
}

func (*Counter) PendingPageVisitRetryScheduled(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	time.Duration,
) {
}
