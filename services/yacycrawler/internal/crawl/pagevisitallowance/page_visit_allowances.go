// Package pagevisitallowance takes for a pending page visit a page of its host,
// another deferral, or another attempt while its limits leave room, and names the
// limit that stopped it when none is left.
package pagevisitallowance

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisitdrop"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

type PageVisitLimits interface {
	TakePageOfHost(
		tx *vault.Txn,
		crawlID crawlid.CrawlID,
		url canonicalurl.CanonicalURL,
		maxPages int,
	) (bool, error)
	TakeDeferral(
		tx *vault.Txn,
		crawlID crawlid.CrawlID,
		url canonicalurl.CanonicalURL,
	) (bool, error)
	TakeAttempt(
		tx *vault.Txn,
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
	storage    *vault.Vault
	limits     PageVisitLimits
	retryDelay retrydelay.Bounds
}

func New(storage *vault.Vault, limits PageVisitLimits, retryDelay retrydelay.Bounds) *Allowances {
	return &Allowances{storage: storage, limits: limits, retryDelay: retryDelay}
}

func (allowances *Allowances) TakePageOfHost(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
	maxPages int,
) (Allowance, error) {
	var took bool
	if err := allowances.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		took, err = allowances.limits.TakePageOfHost(tx, pageVisit.CrawlID, pageVisit.URL, maxPages)
		return err
	}); err != nil {
		return Allowance{}, fmt.Errorf(
			"take an allowance of the page visit to %s: %w",
			pageVisit.URL,
			err,
		)
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
	var took bool
	if err := allowances.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		took, err = allowances.limits.TakeDeferral(tx, pageVisit.CrawlID, pageVisit.URL)
		return err
	}); err != nil {
		return Allowance{}, fmt.Errorf(
			"take an allowance of the page visit to %s: %w",
			pageVisit.URL,
			err,
		)
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
	var attempt int
	var took bool
	if err := allowances.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		attempt, took, err = allowances.limits.TakeAttempt(tx, pageVisit.CrawlID, pageVisit.URL)
		return err
	}); err != nil {
		return Allowance{}, fmt.Errorf(
			"take an allowance of the page visit to %s: %w",
			pageVisit.URL,
			err,
		)
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
