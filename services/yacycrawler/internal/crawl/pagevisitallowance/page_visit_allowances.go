// Package pagevisitallowance takes for a pending page visit a page of its host,
// another deferral, or another attempt while its limits leave room, and names the
// limit that stopped it when none is left.
package pagevisitallowance

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

type PageVisitLimits interface {
	TakePageOfHost(
		ctx context.Context,
		crawlID crawlid.CrawlID,
		url canonicalurl.CanonicalURL,
		host string,
		maxPages int,
	) (bool, error)
	TakeDeferral(
		ctx context.Context,
		crawlID crawlid.CrawlID,
		url canonicalurl.CanonicalURL,
	) (bool, error)
	TakeAttempt(
		ctx context.Context,
		crawlID crawlid.CrawlID,
		url canonicalurl.CanonicalURL,
	) (int, bool, error)
}

type Allowance struct {
	Granted    bool
	PauseFor   time.Duration
	DropReason pagevisitdrop.Reason
}

type Allowances struct {
	limits     PageVisitLimits
	retryDelay retrydelay.Bounds
}

func New(limits PageVisitLimits, retryDelay retrydelay.Bounds) *Allowances {
	return &Allowances{limits: limits, retryDelay: retryDelay}
}

func (allowances *Allowances) TakePageOfHost(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	maxPages int,
) (Allowance, error) {
	took, err := allowances.limits.TakePageOfHost(
		ctx, pageVisit.CrawlID, pageVisit.URL, pageVisit.URL.Hostname(), maxPages,
	)
	if err != nil {
		return Allowance{}, err
	}
	if !took {
		return exhaustedAllowance(pagevisitdrop.HostPagesExhausted), nil
	}
	return grantedAllowance(), nil
}

func (allowances *Allowances) TakeDeferral(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	deferFor time.Duration,
) (Allowance, error) {
	took, err := allowances.limits.TakeDeferral(
		ctx, pageVisit.CrawlID, pageVisit.URL,
	)
	if err != nil {
		return Allowance{}, err
	}
	if !took {
		return exhaustedAllowance(pagevisitdrop.DeferralsExhausted), nil
	}
	return grantedAllowanceAfter(deferFor), nil
}

func (allowances *Allowances) TakeAttempt(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
) (Allowance, error) {
	attempt, took, err := allowances.limits.TakeAttempt(
		ctx, pageVisit.CrawlID, pageVisit.URL,
	)
	if err != nil {
		return Allowance{}, err
	}
	if !took {
		return exhaustedAllowance(pagevisitdrop.RetriesExhausted), nil
	}
	return grantedAllowanceAfter(allowances.retryDelay.Delay(attempt)), nil
}

func grantedAllowance() Allowance {
	return Allowance{Granted: true}
}

func grantedAllowanceAfter(pauseFor time.Duration) Allowance {
	return Allowance{Granted: true, PauseFor: pauseFor}
}

func exhaustedAllowance(reason pagevisitdrop.Reason) Allowance {
	return Allowance{DropReason: reason}
}
