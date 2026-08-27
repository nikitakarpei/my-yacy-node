// Package visitallowance tells a pending visit what it is allowed — the one page
// of its host it holds however often it is redelivered, a deferral or an attempt
// spent after it — and names the allowance that ran out when nothing is left.
package visitallowance

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

type VisitClaims interface {
	Defer(ctx context.Context, orderID string, url canonicalurl.CanonicalURL) (bool, error)
	Retry(ctx context.Context, orderID string, url canonicalurl.CanonicalURL) (int, bool, error)
}

type HostPageAllowances interface {
	HoldsHostPage(
		ctx context.Context,
		orderID string,
		url canonicalurl.CanonicalURL,
		host string,
		maxPages int,
	) (bool, error)
}

type Allowance struct {
	Granted   bool
	Exhausted disposal.Reason
	PauseFor  time.Duration
}

type Ledger struct {
	claims     VisitClaims
	hostPages  HostPageAllowances
	retryDelay retrydelay.Bounds
}

func New(
	claims VisitClaims,
	hostPages HostPageAllowances,
	retryDelay retrydelay.Bounds,
) *Ledger {
	return &Ledger{claims: claims, hostPages: hostPages, retryDelay: retryDelay}
}

func (l *Ledger) HostPageFor(
	ctx context.Context,
	visit pendingvisit.PendingVisit,
	maxPages int,
) (Allowance, error) {
	holdsPage, err := l.hostPages.HoldsHostPage(
		ctx, visit.OrderID, visit.URL, visit.URL.Hostname(), maxPages,
	)
	if err != nil {
		return Allowance{}, fmt.Errorf("hold a host page for %s: %w", visit.URL, err)
	}
	if !holdsPage {
		return exhausted(disposal.HostPagesExhausted), nil
	}
	return Allowance{Granted: true}, nil
}

func (l *Ledger) DeferralFor(
	ctx context.Context,
	visit pendingvisit.PendingVisit,
	deferFor time.Duration,
) (Allowance, error) {
	deferred, err := l.claims.Defer(ctx, visit.OrderID, visit.URL)
	if err != nil {
		return Allowance{}, fmt.Errorf("spend a deferral for %s: %w", visit.URL, err)
	}
	if !deferred {
		return exhausted(disposal.DeferralsExhausted), nil
	}
	return Allowance{Granted: true, PauseFor: deferFor}, nil
}

func (l *Ledger) AttemptFor(
	ctx context.Context,
	visit pendingvisit.PendingVisit,
) (Allowance, error) {
	attempt, retried, err := l.claims.Retry(ctx, visit.OrderID, visit.URL)
	if err != nil {
		return Allowance{}, fmt.Errorf("spend an attempt for %s: %w", visit.URL, err)
	}
	if !retried {
		return exhausted(disposal.RetriesExhausted), nil
	}
	return Allowance{Granted: true, PauseFor: l.retryDelay.Delay(attempt)}, nil
}

func exhausted(reason disposal.Reason) Allowance {
	return Allowance{Exhausted: reason}
}
