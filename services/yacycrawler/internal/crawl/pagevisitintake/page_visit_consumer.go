// Package pagevisitintake pays one pending page visit per lease: it has the page visited
// and ends or returns the visit by what the visit concluded. It leases as many visits as
// it has free workers, and only when its admission lets it.
package pagevisitintake

import (
	"context"
	"errors"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

const returnPause = 30 * time.Second

var errCrawlForgotten = errors.New("the crawl of the page visit is forgotten")

type PendingPageVisits interface {
	Lease(ctx context.Context, amount int) ([]pendingpagevisit.LeasedPageVisit, error)
	End(ctx context.Context, leasedPageVisit pendingpagevisit.LeasedPageVisit)
	Return(
		ctx context.Context,
		leasedPageVisit pendingpagevisit.LeasedPageVisit,
		delay time.Duration,
	)
}

type PageVisitAllowances interface {
	TakePageOfHost(
		ctx context.Context,
		pageVisit pendingpagevisit.PendingPageVisit,
		maxPages int,
	) (pagevisitallowance.Allowance, error)
	TakeDeferral(
		ctx context.Context,
		pageVisit pendingpagevisit.PendingPageVisit,
		deferFor time.Duration,
	) (pagevisitallowance.Allowance, error)
	TakeAttempt(
		ctx context.Context,
		pageVisit pendingpagevisit.PendingPageVisit,
	) (pagevisitallowance.Allowance, error)
}

type Crawls interface {
	OrderOf(
		tx *vault.Txn,
		crawlID crawlid.CrawlID,
	) (order acceptedorder.AcceptedOrder, found bool, err error)
}

type PageVisitConsumer struct {
	storage           *vault.Vault
	pendingPageVisits PendingPageVisits
	allowances        PageVisitAllowances
	crawls            Crawls
	pageVisitor       pagevisit.PageVisitor
	observer          PendingPageVisitObserver
	admission         pullintake.Admission
	fetchConcurrency  int
}

//nolint:revive // a consumer names every collaborator it visits a page with
func NewPageVisitConsumer(
	storage *vault.Vault,
	pendingPageVisits PendingPageVisits,
	allowances PageVisitAllowances,
	crawls Crawls,
	pageVisitor pagevisit.PageVisitor,
	observer PendingPageVisitObserver,
	admission pullintake.Admission,
	fetchConcurrency int,
) *PageVisitConsumer {
	return &PageVisitConsumer{
		storage:           storage,
		pendingPageVisits: pendingPageVisits,
		allowances:        allowances,
		crawls:            crawls,
		pageVisitor:       pageVisitor,
		observer:          observer,
		admission:         admission,
		fetchConcurrency:  fetchConcurrency,
	}
}

func (c *PageVisitConsumer) Run(ctx context.Context) error {
	return pullintake.RunAdmittedBy(
		ctx,
		reportingPageVisitQueue{pendingPageVisits: c.pendingPageVisits, observer: c.observer},
		c.fetchConcurrency,
		c.admission,
		c.payPageVisit,
	)
}

func (c *PageVisitConsumer) payPageVisit(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
) error {
	pageVisit := leasedPageVisit.PageVisit
	order, found, err := c.orderOf(ctx, pageVisit.CrawlID)
	if err != nil {
		c.returnPageVisit(ctx, leasedPageVisit, err)
		return nil
	}
	if !found {
		c.returnPageVisit(ctx, leasedPageVisit, errCrawlForgotten)
		return nil
	}
	if !c.holdsHostPage(ctx, leasedPageVisit, order) {
		return nil
	}
	outcome := c.pageVisitor.VisitPage(ctx, order, pageVisit)
	c.carryOutConclusion(ctx, leasedPageVisit, outcome)
	return nil
}

func (c *PageVisitConsumer) orderOf(
	ctx context.Context,
	crawlID crawlid.CrawlID,
) (acceptedorder.AcceptedOrder, bool, error) {
	var order acceptedorder.AcceptedOrder
	var found bool
	err := c.storage.View(ctx, func(tx *vault.Txn) error {
		var err error
		order, found, err = c.crawls.OrderOf(tx, crawlID)
		return err
	})
	return order, found, err
}

func (c *PageVisitConsumer) returnPageVisit(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	cause error,
) {
	c.observer.PendingPageVisitReturned(ctx, leasedPageVisit.PageVisit, cause)
	c.pendingPageVisits.Return(ctx, leasedPageVisit, returnPause)
}

func (c *PageVisitConsumer) dropPageVisit(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	reason yacycrawlcontract.PageVisitDropReason,
) {
	c.observer.PendingPageVisitDropped(ctx, leasedPageVisit.PageVisit, reason)
	c.pendingPageVisits.End(ctx, leasedPageVisit)
}

func (c *PageVisitConsumer) holdsHostPage(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	order acceptedorder.AcceptedOrder,
) bool {
	allowance, err := c.allowances.TakePageOfHost(
		ctx,
		leasedPageVisit.PageVisit,
		order.MaxPagesPerHost(),
	)
	return c.carryOutAllowance(ctx, leasedPageVisit, allowance, err)
}

func (c *PageVisitConsumer) carryOutAllowance(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	allowance pagevisitallowance.Allowance,
	cause error,
) bool {
	if cause != nil {
		c.returnPageVisit(ctx, leasedPageVisit, cause)
		return false
	}
	if !allowance.Granted {
		c.dropPageVisit(ctx, leasedPageVisit, allowance.DropReason)
		return false
	}
	return true
}

func (c *PageVisitConsumer) carryOutConclusion(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	outcome pagevisit.PageVisitOutcome,
) {
	switch outcome.Conclusion {
	case pagevisit.PageVisitCompleted:
		c.completePageVisit(ctx, leasedPageVisit, outcome.Result)
	case pagevisit.PageVisitDropped:
		c.dropPageVisit(ctx, leasedPageVisit, outcome.DropReason)
	case pagevisit.PageVisitDeferred:
		c.deferPageVisit(ctx, leasedPageVisit, outcome.DeferFor)
	case pagevisit.PageVisitRetryable:
		c.retryPageVisit(ctx, leasedPageVisit)
	case pagevisit.PageVisitReturned:
		c.returnPageVisit(ctx, leasedPageVisit, outcome.ReturnCause)
	}
}

func (c *PageVisitConsumer) completePageVisit(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	result yacycrawlcontract.PageVisitResult,
) {
	c.pendingPageVisits.End(ctx, leasedPageVisit)
	c.observer.PendingPageVisitCompleted(ctx, leasedPageVisit.PageVisit, result)
}

func (c *PageVisitConsumer) deferPageVisit(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
	deferFor time.Duration,
) {
	allowance, err := c.allowances.TakeDeferral(ctx, leasedPageVisit.PageVisit, deferFor)
	if !c.carryOutAllowance(ctx, leasedPageVisit, allowance, err) {
		return
	}
	c.observer.PendingPageVisitDeferred(ctx, leasedPageVisit.PageVisit, allowance.PauseFor)
	c.pendingPageVisits.Return(ctx, leasedPageVisit, allowance.PauseFor)
}

func (c *PageVisitConsumer) retryPageVisit(
	ctx context.Context,
	leasedPageVisit pendingpagevisit.LeasedPageVisit,
) {
	allowance, err := c.allowances.TakeAttempt(ctx, leasedPageVisit.PageVisit)
	if !c.carryOutAllowance(ctx, leasedPageVisit, allowance, err) {
		return
	}
	c.observer.PendingPageVisitRetryScheduled(ctx, leasedPageVisit.PageVisit, allowance.PauseFor)
	c.pendingPageVisits.Return(ctx, leasedPageVisit, allowance.PauseFor)
}
