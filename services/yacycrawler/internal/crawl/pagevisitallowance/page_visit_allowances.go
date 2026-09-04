// Package pagevisitallowance tells a pending page visit what it may still do — take a
// page of its host, another deferral, or another attempt — and names the limit
// that stopped it when it may do none.
package pagevisitallowance

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

type PageVisitLimits interface {
	AdmitsAnotherPageOfHost(
		ctx context.Context,
		orderID string,
		url canonicalurl.CanonicalURL,
		host string,
		maxPages int,
	) (bool, error)
	AdmitsAnotherDeferral(
		ctx context.Context,
		orderID string,
		url canonicalurl.CanonicalURL,
	) (bool, error)
	AdmitsAnotherAttempt(
		ctx context.Context,
		orderID string,
		url canonicalurl.CanonicalURL,
	) (int, bool, error)
}

type Allowance struct {
	Disposal disposal.Reason
	PauseFor time.Duration
}

type Allowances struct {
	limits     PageVisitLimits
	retryDelay retrydelay.Bounds
}

func New(limits PageVisitLimits, retryDelay retrydelay.Bounds) *Allowances {
	return &Allowances{limits: limits, retryDelay: retryDelay}
}

func (allowances *Allowances) HostPageFor(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	maxPages int,
) (Allowance, error) {
	admitted, err := allowances.limits.AdmitsAnotherPageOfHost(
		ctx, pageVisit.OrderID, pageVisit.URL, pageVisit.URL.Hostname(), maxPages,
	)
	if err != nil {
		return Allowance{}, err
	}
	if !admitted {
		return exhaustedAllowance(disposal.HostPagesExhausted), nil
	}
	return grantedAllowance(), nil
}

func (allowances *Allowances) DeferralFor(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	deferFor time.Duration,
) (Allowance, error) {
	admitted, err := allowances.limits.AdmitsAnotherDeferral(
		ctx, pageVisit.OrderID, pageVisit.URL,
	)
	if err != nil {
		return Allowance{}, err
	}
	if !admitted {
		return exhaustedAllowance(disposal.DeferralsExhausted), nil
	}
	return grantedAllowanceAfter(deferFor), nil
}

func (allowances *Allowances) AnotherAttemptFor(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
) (Allowance, error) {
	attempt, admitted, err := allowances.limits.AdmitsAnotherAttempt(
		ctx, pageVisit.OrderID, pageVisit.URL,
	)
	if err != nil {
		return Allowance{}, err
	}
	if !admitted {
		return exhaustedAllowance(disposal.RetriesExhausted), nil
	}
	return grantedAllowanceAfter(allowances.retryDelay.Delay(attempt)), nil
}

func grantedAllowance() Allowance {
	return Allowance{Disposal: disposal.NotDisposed}
}

func grantedAllowanceAfter(pauseFor time.Duration) Allowance {
	return Allowance{Disposal: disposal.NotDisposed, PauseFor: pauseFor}
}

func exhaustedAllowance(reason disposal.Reason) Allowance {
	return Allowance{Disposal: reason}
}
