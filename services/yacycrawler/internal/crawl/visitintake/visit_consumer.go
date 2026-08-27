// Package visitintake pays one pending visit per delivered message: it claims
// the URL for this worker, visits it, and puts the URLs it discovers back on
// the frontier.
package visitintake

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/profileadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/refusal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

const (
	msgOrderUnreadable    = "pending visit names an order the crawler cannot read"
	msgClaimUnreadable    = "visit claim unreadable, the url returns for redelivery"
	msgVisitFailed        = "visit failed, the url returns for redelivery"
	msgDeferralsExhausted = "url dropped after exhausting deferrals"
	msgFetchAbandoned     = "fetch abandoned after retries"
	msgHostPagesSpent     = "url dropped after its host spent its page allowance"
	msgDiscoveryFailed    = "discovered urls not published, the url returns for redelivery"
	msgVisitClaimedTwice  = "pending visit already claimed elsewhere, dropped"

	firstDelivery uint64 = 1
)

type VisitClaims interface {
	Claim(ctx context.Context, orderID string, url canonicalurl.CanonicalURL) (bool, error)
	SpendHostPage(ctx context.Context, orderID string, host string, maxPages int) (bool, error)
	Defer(ctx context.Context, orderID string, url canonicalurl.CanonicalURL) (bool, error)
	Retry(ctx context.Context, orderID string, url canonicalurl.CanonicalURL) (int, bool, error)
}

type AcceptedOrders interface {
	OrderOf(ctx context.Context, orderID string) (yacycrawlcontract.CrawlOrder, error)
}

type PendingVisits interface {
	Publish(ctx context.Context, visit pendingvisit.PendingVisit) error
}

type SettlementProgress interface {
	RefusalHonored(demand refusal.Demand)
	PageDisposed(reason disposal.Reason)
}

type Config struct {
	Source           pullintake.MessageSource
	Claims           VisitClaims
	Orders           AcceptedOrders
	Visits           PendingVisits
	VisitorSource    pagevisit.VisitorSource
	Observer         SettlementProgress
	RetryDelay       retrydelay.Bounds
	FetchConcurrency int
}

type VisitConsumer struct {
	source           pullintake.MessageSource
	claims           VisitClaims
	orders           AcceptedOrders
	visits           PendingVisits
	visitorSource    pagevisit.VisitorSource
	observer         SettlementProgress
	retryDelay       retrydelay.Bounds
	fetchConcurrency int
}

func NewVisitConsumer(config Config) *VisitConsumer {
	return &VisitConsumer{
		source:           config.Source,
		claims:           config.Claims,
		orders:           config.Orders,
		visits:           config.Visits,
		visitorSource:    config.VisitorSource,
		observer:         config.Observer,
		retryDelay:       config.RetryDelay,
		fetchConcurrency: config.FetchConcurrency,
	}
}

func (c *VisitConsumer) Run(ctx context.Context) error {
	return pullintake.Run(ctx, c.source, c.fetchConcurrency, c.processOne)
}

func (c *VisitConsumer) processOne(
	ctx context.Context,
	message pullintake.PendingMessage,
) error {
	visit, err := pendingvisit.UnmarshalPendingVisit(message.Body())
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	order, ordered := c.orderOf(ctx, message, visit)
	if !ordered {
		return nil
	}
	admission, err := profileadmission.New(order.Profile, order.SeedURLs)
	if err != nil {
		return poisonhalt.Halt(ctx, message.Identity(), err)
	}
	if !c.claim(ctx, message, order.Profile, visit) {
		return nil
	}
	outcome, visited := c.visit(ctx, message, order.Profile, visit)
	if !visited {
		return nil
	}
	c.settle(ctx, message, admission, visit, outcome)
	return nil
}

func (c *VisitConsumer) orderOf(
	ctx context.Context,
	message pullintake.PendingMessage,
	visit pendingvisit.PendingVisit,
) (yacycrawlcontract.CrawlOrder, bool) {
	order, err := c.orders.OrderOf(ctx, visit.OrderID)
	if err != nil {
		slog.WarnContext(ctx, msgOrderUnreadable,
			slog.String("order", visit.OrderID),
			slog.String("url", visit.URL.String()),
			slog.Any("error", err),
		)
		message.Return(ctx)
		return yacycrawlcontract.CrawlOrder{}, false
	}
	return order, true
}

func (c *VisitConsumer) claim(
	ctx context.Context,
	message pullintake.PendingMessage,
	profile yacycrawlcontract.CrawlProfile,
	visit pendingvisit.PendingVisit,
) bool {
	claimed, err := c.claims.Claim(ctx, visit.OrderID, visit.URL)
	if err != nil {
		c.returnAfterClaimError(ctx, message, visit, err)
		return false
	}
	if !claimed {
		return c.redelivered(ctx, message, visit)
	}
	return c.spendHostPage(ctx, message, profile, visit)
}

func (c *VisitConsumer) redelivered(
	ctx context.Context,
	message pullintake.PendingMessage,
	visit pendingvisit.PendingVisit,
) bool {
	if message.Deliveries() > firstDelivery {
		return true
	}
	slog.DebugContext(ctx, msgVisitClaimedTwice, slog.String("url", visit.URL.String()))
	message.Acknowledge(ctx)
	return false
}

func (c *VisitConsumer) spendHostPage(
	ctx context.Context,
	message pullintake.PendingMessage,
	profile yacycrawlcontract.CrawlProfile,
	visit pendingvisit.PendingVisit,
) bool {
	spent, err := c.claims.SpendHostPage(
		ctx, visit.OrderID, visit.URL.Hostname(), profile.MaxPagesPerHost,
	)
	if err != nil {
		c.returnAfterClaimError(ctx, message, visit, err)
		return false
	}
	if !spent {
		slog.WarnContext(ctx, msgHostPagesSpent,
			slog.String("url", visit.URL.String()),
			slog.Int("maxPagesPerHost", profile.MaxPagesPerHost),
		)
		c.observer.PageDisposed(disposal.HostPagesSpent)
		message.Acknowledge(ctx)
		return false
	}
	return true
}

func (c *VisitConsumer) returnAfterClaimError(
	ctx context.Context,
	message pullintake.PendingMessage,
	visit pendingvisit.PendingVisit,
	cause error,
) {
	slog.WarnContext(ctx, msgClaimUnreadable,
		slog.String("url", visit.URL.String()),
		slog.Any("error", cause),
	)
	message.Return(ctx)
}

func (c *VisitConsumer) visit(
	ctx context.Context,
	message pullintake.PendingMessage,
	profile yacycrawlcontract.CrawlProfile,
	visit pendingvisit.PendingVisit,
) (pagevisit.VisitOutcome, bool) {
	visitor := c.visitorSource.VisitorFor(indexingRefusalOf(profile))
	outcome, err := visitor.Visit(ctx, visit.URL)
	if err != nil {
		slog.WarnContext(ctx, msgVisitFailed,
			slog.String("url", visit.URL.String()),
			slog.Any("error", err),
		)
		message.Return(ctx)
		return pagevisit.VisitOutcome{}, false
	}
	return outcome, true
}

func indexingRefusalOf(profile yacycrawlcontract.CrawlProfile) pagevisit.IndexingRefusal {
	if profile.IgnoresIndexingRefusal {
		return pagevisit.Ignored
	}
	return pagevisit.Honored
}

func (c *VisitConsumer) settle(
	ctx context.Context,
	message pullintake.PendingMessage,
	admission *profileadmission.Admission,
	visit pendingvisit.PendingVisit,
	outcome pagevisit.VisitOutcome,
) {
	switch outcome.Conclusion {
	case pagevisit.VisitDeferred:
		c.settleDeferred(ctx, message, visit, outcome.DeferFor)
	case pagevisit.VisitRetryable:
		c.settleRetryable(ctx, message, visit)
	case pagevisit.VisitCompleted:
		c.settleCompleted(ctx, message, admission, visit, outcome)
	}
}

func (c *VisitConsumer) settleDeferred(
	ctx context.Context,
	message pullintake.PendingMessage,
	visit pendingvisit.PendingVisit,
	deferFor time.Duration,
) {
	deferred, err := c.claims.Defer(ctx, visit.OrderID, visit.URL)
	if err != nil {
		c.returnAfterClaimError(ctx, message, visit, err)
		return
	}
	if !deferred {
		slog.WarnContext(ctx, msgDeferralsExhausted, slog.String("url", visit.URL.String()))
		c.observer.PageDisposed(disposal.DeferralsExhausted)
		message.Acknowledge(ctx)
		return
	}
	c.observer.RefusalHonored(refusal.Defer)
	message.ReturnAfter(ctx, deferFor)
}

func (c *VisitConsumer) settleRetryable(
	ctx context.Context,
	message pullintake.PendingMessage,
	visit pendingvisit.PendingVisit,
) {
	attempt, retried, err := c.claims.Retry(ctx, visit.OrderID, visit.URL)
	if err != nil {
		c.returnAfterClaimError(ctx, message, visit, err)
		return
	}
	if !retried {
		slog.WarnContext(ctx, msgFetchAbandoned, slog.String("url", visit.URL.String()))
		c.observer.PageDisposed(disposal.FetchAbandoned)
		message.Acknowledge(ctx)
		return
	}
	message.ReturnAfter(ctx, c.retryDelay.Delay(attempt))
}

func (c *VisitConsumer) settleCompleted(
	ctx context.Context,
	message pullintake.PendingMessage,
	admission *profileadmission.Admission,
	visit pendingvisit.PendingVisit,
	outcome pagevisit.VisitOutcome,
) {
	if outcome.Disposal != disposal.NotDisposed {
		c.observer.PageDisposed(outcome.Disposal)
	}
	if err := c.discover(ctx, admission, visit, outcome.DiscoveredURLs); err != nil {
		slog.WarnContext(ctx, msgDiscoveryFailed,
			slog.String("url", visit.URL.String()),
			slog.Any("error", err),
		)
		message.Return(ctx)
		return
	}
	message.Acknowledge(ctx)
}

func (c *VisitConsumer) discover(
	ctx context.Context,
	admission *profileadmission.Admission,
	visit pendingvisit.PendingVisit,
	discoveredURLs []canonicalurl.CanonicalURL,
) error {
	depth := visit.Depth + 1
	for _, url := range discoveredURLs {
		if !admission.Admits(url, depth) {
			continue
		}
		if err := c.visits.Publish(ctx, pendingvisit.PendingVisit{
			OrderID: visit.OrderID,
			URL:     url,
			Depth:   depth,
		}); err != nil {
			return err
		}
	}
	return nil
}
