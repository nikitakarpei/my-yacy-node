// Package pagevisitlimits holds what a crawl has already used for one page visit — a
// page of the URL's host, the deferrals of that URL and its attempts — and admits another
// only while the crawl's limit leaves room for it. A redelivered page visit takes the page
// of its host once.
package pagevisitlimits

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/urlhash"
)

type MaxPerURL struct {
	Deferrals int
	Attempts  int
}

type PageVisitLimits struct {
	takenHostPageCounts *vault.ExpiringCollection[hostKey, takenHostPageCount]
	usedAllowances      *vault.ExpiringCollection[pageVisitKey, usedAllowances]
	maxPerURL           MaxPerURL
}

func Open(
	ctx context.Context,
	storage *vault.Vault,
	keptFor time.Duration,
	maxPerURL MaxPerURL,
) (*PageVisitLimits, error) {
	takenHostPageCounts, usedAllowances, err := registerPageVisitLimits(ctx, storage, keptFor)
	if err != nil {
		return nil, err
	}
	return &PageVisitLimits{
		takenHostPageCounts: takenHostPageCounts,
		usedAllowances:      usedAllowances,
		maxPerURL:           maxPerURL,
	}, nil
}

type hostKey struct {
	crawlID crawlid.CrawlID
	host    string
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	url     urlhash.URLHash
}

type takenHostPageCount struct {
	pages int
}

type usedAllowances struct {
	pageOfHost bool
	deferrals  int
	attempts   int
}

func (limits *PageVisitLimits) TakePageOfHost(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	maxPages int,
) (bool, error) {
	if maxPages == yacycrawlcontract.UnlimitedPagesPerHost {
		return true, nil
	}
	took, err := limits.takePageOfHost(
		tx,
		pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)},
		hostKey{crawlID: crawlID, host: url.Hostname()},
		maxPages,
	)
	if err != nil {
		return false, fmt.Errorf("take a page of host %s for %s: %w", url.Hostname(), url, err)
	}
	return took, nil
}

func (limits *PageVisitLimits) takePageOfHost(
	tx *vault.Txn,
	pageVisit pageVisitKey,
	crawlHost hostKey,
	maxPages int,
) (bool, error) {
	used, _, err := limits.usedAllowances.Get(tx, pageVisit)
	if err != nil || used.pageOfHost {
		return used.pageOfHost, err
	}
	pageCount, _, err := limits.takenHostPageCounts.Get(tx, crawlHost)
	if err != nil || pageCount.pages >= maxPages {
		return false, err
	}
	pageCount.pages++
	if err := limits.takenHostPageCounts.Put(tx, crawlHost, pageCount); err != nil {
		return false, err
	}
	used.pageOfHost = true
	return true, limits.usedAllowances.Put(tx, pageVisit, used)
}

func (limits *PageVisitLimits) TakeDeferral(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	_, deferred, err := limits.takeAllowance(tx, crawlID, url, limits.withAnotherDeferral)
	if err != nil {
		return false, fmt.Errorf("take a deferral of the page visit to %s: %w", url, err)
	}
	return deferred, nil
}

func (limits *PageVisitLimits) takeAllowance(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	withAnother func(usedAllowances) (usedAllowances, bool),
) (usedAllowances, bool, error) {
	key := pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)}
	used, _, err := limits.usedAllowances.Get(tx, key)
	if err != nil {
		return usedAllowances{}, false, err
	}
	standing, took := withAnother(used)
	if !took {
		return standing, false, nil
	}
	return standing, true, limits.usedAllowances.Put(tx, key, standing)
}

func (limits *PageVisitLimits) withAnotherDeferral(used usedAllowances) (usedAllowances, bool) {
	if used.deferrals >= limits.maxPerURL.Deferrals {
		return used, false
	}
	used.deferrals++
	return used, true
}

func (limits *PageVisitLimits) TakeAttempt(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (int, bool, error) {
	standing, attempted, err := limits.takeAllowance(tx, crawlID, url, limits.withAnotherAttempt)
	if err != nil {
		return 0, false, fmt.Errorf("take an attempt of the page visit to %s: %w", url, err)
	}
	return standing.attempts, attempted, nil
}

func (limits *PageVisitLimits) withAnotherAttempt(used usedAllowances) (usedAllowances, bool) {
	if used.attempts >= limits.maxPerURL.Attempts {
		return used, false
	}
	used.attempts++
	return used, true
}
