// Package visitintake pays one pending visit per delivered message: it claims
// the URL for that message, visits it, and puts the URLs it discovers back on
// the frontier.
package visitintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/visitclaim"
)

const (
	msgVisitReturned      = "pending visit returned for redelivery"
	msgAllowanceExhausted = "pending visit dropped after exhausting its allowance"
	msgClaimHeldElsewhere = "pending visit already claimed elsewhere, dropped"
)

type VisitClaims interface {
	Claim(
		ctx context.Context,
		orderID string,
		url canonicalurl.CanonicalURL,
		holder string,
	) (visitclaim.Claim, error)
	SpendHostPage(
		ctx context.Context,
		orderID string,
		url canonicalurl.CanonicalURL,
		host string,
		maxPages int,
	) (bool, error)
	Defer(ctx context.Context, orderID string, url canonicalurl.CanonicalURL) (bool, error)
	Retry(ctx context.Context, orderID string, url canonicalurl.CanonicalURL) (int, bool, error)
}

type AcceptedOrders interface {
	OrderOf(ctx context.Context, orderID string) (acceptedorder.AcceptedOrder, error)
}

type PendingVisits interface {
	Publish(ctx context.Context, visit pendingvisit.PendingVisit) error
}

type PendingVisitProgress interface {
	RefusalHonored(demand refusal.Demand)
	PageDisposed(reason disposal.Reason)
}

type VisitConsumer struct {
	source           pullintake.MessageSource
	claims           VisitClaims
	orders           AcceptedOrders
	frontier         PendingVisits
	visitorFor       pagevisit.VisitorFor
	observer         PendingVisitProgress
	retryDelay       retrydelay.Bounds
	fetchConcurrency int
}

//nolint:revive // a consumer names every collaborator it pays a visit with
func NewVisitConsumer(
	source pullintake.MessageSource,
	claims VisitClaims,
	orders AcceptedOrders,
	frontier PendingVisits,
	visitorFor pagevisit.VisitorFor,
	observer PendingVisitProgress,
	retryDelay retrydelay.Bounds,
	fetchConcurrency int,
) *VisitConsumer {
	return &VisitConsumer{
		source:           source,
		claims:           claims,
		orders:           orders,
		frontier:         frontier,
		visitorFor:       visitorFor,
		observer:         observer,
		retryDelay:       retryDelay,
		fetchConcurrency: fetchConcurrency,
	}
}

func (c *VisitConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.fetchConcurrency, c.payVisit)
}

func (c *VisitConsumer) payVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	pendingVisit, err := pendingvisit.UnmarshalPendingVisit(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	order, err := c.orders.OrderOf(ctx, pendingVisit.OrderID)
	if err != nil {
		c.returnVisit(ctx, message, pendingVisit, err)
		return nil
	}
	if !c.claimVisit(ctx, message, order, pendingVisit) {
		return nil
	}
	outcome, err := c.visitorFor(ignoredRefusalsOf(order)).Visit(ctx, pendingVisit.URL)
	if err != nil {
		c.returnVisit(ctx, message, pendingVisit, err)
		return nil
	}
	c.carryOutConclusion(ctx, message, order, pendingVisit, outcome)
	return nil
}

func (c *VisitConsumer) returnVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingVisit pendingvisit.PendingVisit,
	cause error,
) {
	slog.WarnContext(ctx, msgVisitReturned,
		slog.String("order", pendingVisit.OrderID),
		slog.String("url", pendingVisit.URL.String()),
		slog.Any("error", cause),
	)
	message.Return(ctx)
}

func (c *VisitConsumer) claimVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingVisit pendingvisit.PendingVisit,
) bool {
	claim, err := c.claims.Claim(
		ctx, pendingVisit.OrderID, pendingVisit.URL, message.Identity(),
	)
	if err != nil {
		c.returnVisit(ctx, message, pendingVisit, err)
		return false
	}
	switch claim {
	case visitclaim.Taken, visitclaim.Resumed:
		return c.spendHostPage(ctx, message, order, pendingVisit)
	}
	c.dropClaimHeldElsewhere(ctx, message, pendingVisit)
	return false
}

func (c *VisitConsumer) dropClaimHeldElsewhere(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingVisit pendingvisit.PendingVisit,
) {
	slog.DebugContext(ctx, msgClaimHeldElsewhere,
		slog.String("order", pendingVisit.OrderID),
		slog.String("url", pendingVisit.URL.String()),
	)
	message.Acknowledge(ctx)
}

func (c *VisitConsumer) spendHostPage(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingVisit pendingvisit.PendingVisit,
) bool {
	spent, err := c.claims.SpendHostPage(
		ctx,
		pendingVisit.OrderID,
		pendingVisit.URL,
		pendingVisit.URL.Hostname(),
		order.MaxPagesPerHost(),
	)
	if err != nil {
		c.returnVisit(ctx, message, pendingVisit, err)
		return false
	}
	if !spent {
		c.dropExhaustedVisit(ctx, message, pendingVisit, disposal.HostPagesExhausted)
		return false
	}
	return true
}

func (c *VisitConsumer) dropExhaustedVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingVisit pendingvisit.PendingVisit,
	allowance disposal.Reason,
) {
	slog.WarnContext(ctx, msgAllowanceExhausted,
		slog.String("order", pendingVisit.OrderID),
		slog.String("url", pendingVisit.URL.String()),
		slog.String("allowance", string(allowance)),
	)
	c.observer.PageDisposed(allowance)
	message.Acknowledge(ctx)
}

func ignoredRefusalsOf(order acceptedorder.AcceptedOrder) pagevisit.IgnoredRefusals {
	return pagevisit.IgnoredRefusals{IndexingRefusal: order.IgnoresIndexingRefusal()}
}

func (c *VisitConsumer) carryOutConclusion(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingVisit pendingvisit.PendingVisit,
	outcome pagevisit.VisitOutcome,
) {
	switch outcome.Conclusion {
	case pagevisit.VisitDeferred:
		c.deferVisit(ctx, message, pendingVisit, outcome.DeferFor)
	case pagevisit.VisitRetryable:
		c.retryVisit(ctx, message, pendingVisit)
	case pagevisit.VisitCompleted:
		c.completeVisit(ctx, message, order, pendingVisit, outcome)
	}
}

func (c *VisitConsumer) deferVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingVisit pendingvisit.PendingVisit,
	deferFor time.Duration,
) {
	deferred, err := c.claims.Defer(ctx, pendingVisit.OrderID, pendingVisit.URL)
	if err != nil {
		c.returnVisit(ctx, message, pendingVisit, err)
		return
	}
	if !deferred {
		c.dropExhaustedVisit(ctx, message, pendingVisit, disposal.DeferralsExhausted)
		return
	}
	c.observer.RefusalHonored(refusal.Defer)
	message.ReturnAfter(ctx, deferFor)
}

func (c *VisitConsumer) retryVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	pendingVisit pendingvisit.PendingVisit,
) {
	attempt, retried, err := c.claims.Retry(ctx, pendingVisit.OrderID, pendingVisit.URL)
	if err != nil {
		c.returnVisit(ctx, message, pendingVisit, err)
		return
	}
	if !retried {
		c.dropExhaustedVisit(ctx, message, pendingVisit, disposal.RetriesExhausted)
		return
	}
	message.ReturnAfter(ctx, c.retryDelay.Delay(attempt))
}

func (c *VisitConsumer) completeVisit(
	ctx context.Context,
	message pullintake.PendingMessage,
	order acceptedorder.AcceptedOrder,
	pendingVisit pendingvisit.PendingVisit,
	outcome pagevisit.VisitOutcome,
) {
	if outcome.Disposed() {
		c.observer.PageDisposed(outcome.Disposal)
	}
	if err := c.putDiscoveredURLsOnFrontier(
		ctx,
		order,
		pendingVisit,
		outcome.DiscoveredURLs,
	); err != nil {
		c.returnVisit(ctx, message, pendingVisit, err)
		return
	}
	message.Acknowledge(ctx)
}

func (c *VisitConsumer) putDiscoveredURLsOnFrontier(
	ctx context.Context,
	order acceptedorder.AcceptedOrder,
	pendingVisit pendingvisit.PendingVisit,
	discoveredURLs []canonicalurl.CanonicalURL,
) error {
	depth := pendingVisit.Depth + 1
	for _, url := range discoveredURLs {
		if !order.Admits(url, depth) {
			continue
		}
		if err := c.frontier.Publish(ctx, pendingvisit.PendingVisit{
			OrderID: pendingVisit.OrderID,
			URL:     url,
			Depth:   depth,
		}); err != nil {
			return err
		}
	}
	return nil
}
