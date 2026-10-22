package crawlprogress

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type CountingObserver interface {
	CrawlProgressCountFailed(ctx context.Context, crawlID crawlid.CrawlID, count string, err error)
}

type Counting struct {
	storage  *vault.Vault
	counts   *Counts
	observer CountingObserver
}

func NewCounting(storage *vault.Vault, counts *Counts, observer CountingObserver) *Counting {
	return &Counting{storage: storage, counts: counts, observer: observer}
}

func (counting *Counting) PendingPageVisitCompleted(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	result yacycrawlcontract.PageVisitResult,
) {
	crawlID := pendingPageVisit.CrawlID
	counting.count(ctx, crawlID, completedCountOf(result), func(tx *vault.Txn) error {
		return counting.counts.CountCompleted(tx, crawlID, pendingPageVisit.URL.Hostname(), result)
	})
}

func (counting *Counting) PendingPageVisitDropped(
	ctx context.Context,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	reason yacycrawlcontract.PageVisitDropReason,
) {
	crawlID := pendingPageVisit.CrawlID
	counting.count(ctx, crawlID, droppedCountOf(reason), func(tx *vault.Txn) error {
		return counting.counts.CountDropped(tx, crawlID, pendingPageVisit.URL.Hostname(), reason, 1)
	})
}

func (counting *Counting) LinksUnadmitted(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	amountOfLinks int,
) {
	counting.count(ctx, crawlID, unadmittedLinksCount, func(tx *vault.Txn) error {
		return counting.counts.CountUnadmittedLinks(tx, crawlID, uint(amountOfLinks))
	})
}

func (counting *Counting) count(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	count string,
	raise func(tx *vault.Txn) error,
) {
	if err := counting.storage.Update(ctx, raise); err != nil {
		counting.observer.CrawlProgressCountFailed(ctx, crawlID, count, err)
	}
}

func (*Counting) PendingPageVisitReturned(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	error,
) {
}

func (*Counting) PendingPageVisitDeferred(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	time.Duration,
) {
}

func (*Counting) PendingPageVisitRetryScheduled(
	context.Context,
	pendingpagevisit.PendingPageVisit,
	time.Duration,
) {
}
