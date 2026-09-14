// Package pagevisitintake pays one pending page visit per delivered message: it claims
// the URL for that message, visits it, and puts the URLs it discovers back on
// the frontier.
package pagevisitintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitallowance"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type TakenPageVisits interface {
	TakePageVisit(
		ctx context.Context,
		orderID string,
		url canonicalurl.CanonicalURL,
		taker string,
	) (bool, error)
}

type PageVisitAllowances interface {
	HostPageFor(
		ctx context.Context,
		pageVisit pendingpagevisit.PendingPageVisit,
		maxPages int,
	) (pagevisitallowance.Allowance, error)
	DeferralFor(
		ctx context.Context,
		pageVisit pendingpagevisit.PendingPageVisit,
		deferFor time.Duration,
	) (pagevisitallowance.Allowance, error)
	AnotherAttemptFor(
		ctx context.Context,
		pageVisit pendingpagevisit.PendingPageVisit,
	) (pagevisitallowance.Allowance, error)
}

type AcceptedOrders interface {
	OrderOf(ctx context.Context, orderID string) (acceptedorder.AcceptedOrder, error)
}

type PendingPageVisits interface {
	Publish(ctx context.Context, pageVisit pendingpagevisit.PendingPageVisit) error
}

type PageVisitConsumer struct {
	source           pullintake.MessageSource
	takenPageVisits  TakenPageVisits
	allowances       PageVisitAllowances
	orders           AcceptedOrders
	frontier         PendingPageVisits
	pageVisitor      pagevisit.PageVisitor
	observer         PendingPageVisitObserver
	fetchConcurrency int
}

//nolint:revive // a consumer names every collaborator it visits a page with
func NewPageVisitConsumer(
	source pullintake.MessageSource,
	takenPageVisits TakenPageVisits,
	allowances PageVisitAllowances,
	orders AcceptedOrders,
	frontier PendingPageVisits,
	pageVisitor pagevisit.PageVisitor,
	observer PendingPageVisitObserver,
	fetchConcurrency int,
) *PageVisitConsumer {
	return &PageVisitConsumer{
		source:           source,
		takenPageVisits:  takenPageVisits,
		allowances:       allowances,
		orders:           orders,
		frontier:         frontier,
		pageVisitor:      pageVisitor,
		observer:         observer,
		fetchConcurrency: fetchConcurrency,
	}
}

func (c *PageVisitConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.fetchConcurrency, c.payPageVisit)
}

func (c *PageVisitConsumer) payPageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	pendingPageVisit, err := pendingpagevisit.UnmarshalPendingPageVisit(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	order, err := c.orders.OrderOf(ctx, pendingPageVisit.OrderID)
	if err != nil {
		c.returnPageVisit(ctx, message, pendingPageVisit, err)
		return nil
	}
	if !c.takePageVisit(ctx, message, order, pendingPageVisit) {
		return nil
	}
	outcome := c.pageVisitor.VisitPage(ctx, pendingPageVisit.URL)
	c.carryOutConclusion(ctx, message, order, pendingPageVisit, outcome)
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
		ctx, pendingPageVisit.OrderID, pendingPageVisit.URL, message.Identity(),
	)
	if err != nil {
		c.returnPageVisit(ctx, message, pendingPageVisit, err)
		return false
	}
	if !took {
		c.dropPageVisitTakenByAnother(ctx, message, pendingPageVisit)
		return false
	}
	return c.holdsHostPage(ctx, message, order, pendingPageVisit)
}

func (c *PageVisitConsumer) dropPageVisitTakenByAnother(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) {
	c.observer.PendingPageVisitDroppedAsTakenByAnother(ctx, pendingPageVisit)
	message.Acknowledge(ctx)
}

func (c *PageVisitConsumer) holdsHostPage(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
) bool {
	allowance, err := c.allowances.HostPageFor(ctx, pendingPageVisit, order.MaxPagesPerHost())
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
	if allowance.Disposal.DisposedThePage() {
		c.dropExhaustedPageVisit(ctx, message, pendingPageVisit, allowance.Disposal)
		return false
	}
	return true
}

func (c *PageVisitConsumer) dropExhaustedPageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	reason disposal.Reason,
) {
	c.observer.PendingPageVisitDisposedPage(ctx, pendingPageVisit, reason)
	message.Acknowledge(ctx)
}

func (c *PageVisitConsumer) carryOutConclusion(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	outcome pagevisit.PageVisitOutcome,
) {
	switch outcome.Conclusion {
	case pagevisit.PageVisitDeferred:
		c.deferPageVisit(ctx, message, pendingPageVisit, outcome.DeferFor)
	case pagevisit.PageVisitRetryable:
		c.retryPageVisit(ctx, message, pendingPageVisit)
	case pagevisit.PageVisitTerminal:
		c.completePageVisit(ctx, message, order, pendingPageVisit, outcome)
	}
}

func (c *PageVisitConsumer) deferPageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	deferFor time.Duration,
) {
	allowance, err := c.allowances.DeferralFor(ctx, pendingPageVisit, deferFor)
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
	allowance, err := c.allowances.AnotherAttemptFor(ctx, pendingPageVisit)
	if !c.carryOutAllowance(ctx, message, pendingPageVisit, allowance, err) {
		return
	}
	c.observer.PendingPageVisitRetryScheduled(ctx, pendingPageVisit, allowance.PauseFor)
	message.ReturnAfter(ctx, allowance.PauseFor)
}

func (c *PageVisitConsumer) completePageVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	outcome pagevisit.PageVisitOutcome,
) {
	if outcome.Disposal.DisposedThePage() {
		c.observer.PendingPageVisitDisposedPage(ctx, pendingPageVisit, outcome.Disposal)
	}
	if err := c.putDiscoveredURLsOnFrontier(
		ctx,
		order,
		pendingPageVisit,
		outcome.DiscoveredURLs,
	); err != nil {
		c.returnPageVisit(ctx, message, pendingPageVisit, err)
		return
	}
	if err := c.putRedirectTargetOnFrontier(
		ctx,
		order,
		pendingPageVisit,
		outcome.RedirectTarget,
	); err != nil {
		c.returnPageVisit(ctx, message, pendingPageVisit, err)
		return
	}
	message.Acknowledge(ctx)
	c.observer.PendingPageVisitCompleted(ctx, pendingPageVisit)
}

func (c *PageVisitConsumer) putRedirectTargetOnFrontier(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	redirectTarget canonicalurl.CanonicalURL,
) error {
	if redirectTarget.String() == "" {
		return nil
	}
	if !order.Admits(redirectTarget, pendingPageVisit.Depth) {
		return nil
	}
	return c.frontier.Publish(ctx, pendingpagevisit.PendingPageVisit{
		OrderID: pendingPageVisit.OrderID,
		URL:     redirectTarget,
		Depth:   pendingPageVisit.Depth,
	})
}

func (c *PageVisitConsumer) putDiscoveredURLsOnFrontier(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
	pendingPageVisit pendingpagevisit.PendingPageVisit,
	discoveredURLs []canonicalurl.CanonicalURL,
) error {
	depth := pendingPageVisit.Depth + 1
	for _, url := range discoveredURLs {
		if !order.Admits(url, depth) {
			continue
		}
		if err := c.frontier.Publish(ctx, pendingpagevisit.PendingPageVisit{
			OrderID: pendingPageVisit.OrderID,
			URL:     url,
			Depth:   depth,
		}); err != nil {
			return err
		}
	}
	return nil
}
