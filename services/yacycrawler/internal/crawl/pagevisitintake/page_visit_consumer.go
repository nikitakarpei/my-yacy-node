// Package pagevisitintake pays one pending page visit per delivered message: it claims
// the URL for that message, has the page visited, and settles the message by what the
// visit concluded. It takes the next message only when its admission lets it.
package pagevisitintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type TakenPageVisits interface {
	TakePageVisit(
		ctx context.Context,
		crawlID crawlid.CrawlID,
		url canonicalurl.CanonicalURL,
		taker string,
	) (bool, error)
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
		ctx context.Context,
		crawlID crawlid.CrawlID,
	) (order acceptedorder.AcceptedOrder, found bool, err error)
}

type PageVisitConsumer struct {
	pendingPageVisits pullintake.PendingMessages
	takenPageVisits   TakenPageVisits
	allowances        PageVisitAllowances
	crawls            Crawls
	pageVisitor       pagevisit.PageVisitor
	observer          PendingPageVisitObserver
	admission         pullintake.Admission
	fetchConcurrency  int
}

//nolint:revive // a consumer names every collaborator it visits a page with
func NewPageVisitConsumer(
	pendingPageVisits pullintake.PendingMessages,
	takenPageVisits TakenPageVisits,
	allowances PageVisitAllowances,
	crawls Crawls,
	pageVisitor pagevisit.PageVisitor,
	observer PendingPageVisitObserver,
	admission pullintake.Admission,
	fetchConcurrency int,
) *PageVisitConsumer {
	return &PageVisitConsumer{
		pendingPageVisits: pendingPageVisits,
		takenPageVisits:   takenPageVisits,
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
		ctx, c.pendingPageVisits, c.fetchConcurrency, c.admission, c.payPageVisit,
	)
}

func (c *PageVisitConsumer) payPageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	pendingPageVisit, err := pendingpagevisit.UnmarshalPendingPageVisit(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	order, found, err := c.crawls.OrderOf(ctx, pendingPageVisit.CrawlID)
	if err != nil {
		c.returnPageVisit(ctx, message, pendingPageVisit, err)
		return nil
	}
	if !found {
		c.dropPageVisit(ctx, message, pendingPageVisit, pagevisitdrop.OrderForgotten)
		return nil
	}
	if !c.takePageVisit(ctx, message, order, pendingPageVisit) {
		return nil
	}
	outcome := c.pageVisitor.VisitPage(ctx, order, pendingPageVisit)
	c.carryOutConclusion(ctx, message, pendingPageVisit, outcome)
	return nil
}

func (c *PageVisitConsumer) returnPageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	cause error,
) {
	c.observer.PendingPageVisitReturned(ctx, pendingPageVisit, cause)
	message.Return(ctx)
}

func (c *PageVisitConsumer) takePageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) bool {
	took, err := c.takenPageVisits.TakePageVisit(
		ctx, pendingPageVisit.CrawlID, pendingPageVisit.URL, message.Identity(),
	)
	if err != nil {
		c.returnPageVisit(ctx, message, pendingPageVisit, err)
		return false
	}
	if !took {
		c.dropPageVisit(ctx, message, pendingPageVisit, pagevisitdrop.TakenByAnother)
		return false
	}
	return c.holdsHostPage(ctx, message, order, pendingPageVisit)
}

func (c *PageVisitConsumer) dropPageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	reason pagevisitdrop.Reason,
) {
	c.observer.PendingPageVisitDropped(ctx, pendingPageVisit, reason)
	message.Acknowledge(ctx)
}

func (c *PageVisitConsumer) holdsHostPage(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) bool {
	allowance, err := c.allowances.TakePageOfHost(ctx, pendingPageVisit, order.MaxPagesPerHost())
	return c.carryOutAllowance(ctx, message, pendingPageVisit, allowance, err)
}

func (c *PageVisitConsumer) carryOutAllowance(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	allowance pagevisitallowance.Allowance,
	cause error,
) bool {
	if cause != nil {
		c.returnPageVisit(ctx, message, pendingPageVisit, cause)
		return false
	}
	if !allowance.Granted {
		c.dropPageVisit(ctx, message, pendingPageVisit, allowance.DropReason)
		return false
	}
	return true
}

func (c *PageVisitConsumer) carryOutConclusion(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	outcome pagevisit.PageVisitOutcome,
) {
	switch outcome.Conclusion {
	case pagevisit.PageVisitCompleted:
		c.completePageVisit(ctx, message, pendingPageVisit, outcome.Result)
	case pagevisit.PageVisitDropped:
		c.dropPageVisit(ctx, message, pendingPageVisit, outcome.DropReason)
	case pagevisit.PageVisitDeferred:
		c.deferPageVisit(ctx, message, pendingPageVisit, outcome.DeferFor)
	case pagevisit.PageVisitRetryable:
		c.retryPageVisit(ctx, message, pendingPageVisit)
	case pagevisit.PageVisitReturned:
		c.returnPageVisit(ctx, message, pendingPageVisit, outcome.ReturnCause)
	}
}

func (c *PageVisitConsumer) completePageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	result pagevisitresult.Result,
) {
	message.Acknowledge(ctx)
	c.observer.PendingPageVisitCompleted(ctx, pendingPageVisit, result)
}

func (c *PageVisitConsumer) deferPageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	deferFor time.Duration,
) {
	allowance, err := c.allowances.TakeDeferral(ctx, pendingPageVisit, deferFor)
	if !c.carryOutAllowance(ctx, message, pendingPageVisit, allowance, err) {
		return
	}
	c.observer.PendingPageVisitDeferred(ctx, pendingPageVisit, allowance.PauseFor)
	message.ReturnAfter(ctx, allowance.PauseFor)
}

func (c *PageVisitConsumer) retryPageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) {
	allowance, err := c.allowances.TakeAttempt(ctx, pendingPageVisit)
	if !c.carryOutAllowance(ctx, message, pendingPageVisit, allowance, err) {
		return
	}
	c.observer.PendingPageVisitRetryScheduled(ctx, pendingPageVisit, allowance.PauseFor)
	message.ReturnAfter(ctx, allowance.PauseFor)
}
