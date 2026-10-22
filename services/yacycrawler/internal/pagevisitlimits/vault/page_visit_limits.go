// Package vault holds in a vault what a crawl has already used for one page visit
// — a page of the URL's host, the deferrals of that URL and its attempts — and
// admits another only while the crawl's limit leaves room for it. A redelivered
// page visit takes the page of its host once.
package vault

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

var errLimitMalformed = errors.New("page visit limit record is malformed")

const (
	takenHostPageCountsBucket = "taken_host_page_counts"
	usedAllowancesBucket      = "used_page_visit_allowances"
)

type MaxPerURL struct {
	Deferrals int
	Attempts  int
}

type PageVisitLimits struct {
	storage             *vault.Vault
	takenHostPageCounts *vault.ExpiringCollection[hostKey, takenHostPageCount]
	usedAllowances      *vault.ExpiringCollection[pageVisitKey, usedAllowances]
	maxPerURL           MaxPerURL
}

func New(
	ctx context.Context,
	storage *vault.Vault,
	keptFor time.Duration,
	maxPerURL MaxPerURL,
) (*PageVisitLimits, error) {
	takenHostPageCounts, err := storage.RegisterExpiringCollection(
		ctx, takenHostPageCountsBucket, hostKeys, takenHostPageCountCodec{}, keptFor, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", takenHostPageCountsBucket, err)
	}
	usedAllowances, err := storage.RegisterExpiringCollection(
		ctx, usedAllowancesBucket, pageVisitKeys, usedAllowancesCodec{}, keptFor, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", usedAllowancesBucket, err)
	}
	return &PageVisitLimits{
		storage:             storage,
		takenHostPageCounts: takenHostPageCounts,
		usedAllowances:      usedAllowances,
		maxPerURL:           maxPerURL,
	}, nil
}

func (limits *PageVisitLimits) TakePageOfHost(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	maxPages int,
) (bool, error) {
	if maxPages == yacycrawlcontract.UnlimitedPagesPerHost {
		return true, nil
	}
	pageVisit := pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)}
	crawlHost := hostKey{crawlID: crawlID, host: url.Hostname()}
	var took bool
	if err := limits.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		took, err = limits.takePageOfHost(tx, pageVisit, crawlHost, maxPages)
		return err
	}); err != nil {
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
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	_, deferred, err := limits.takeAllowance(ctx, crawlID, url, limits.withAnotherDeferral)
	if err != nil {
		return false, fmt.Errorf("take a deferral of the page visit to %s: %w", url, err)
	}
	return deferred, nil
}

func (limits *PageVisitLimits) takeAllowance(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	withAnother func(usedAllowances) (usedAllowances, bool),
) (usedAllowances, bool, error) {
	key := pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)}
	var standing usedAllowances
	var took bool
	err := limits.storage.Update(ctx, func(tx *vault.Txn) error {
		used, _, err := limits.usedAllowances.Get(tx, key)
		if err != nil {
			return err
		}
		standing, took = withAnother(used)
		if !took {
			return nil
		}
		return limits.usedAllowances.Put(tx, key, standing)
	})
	return standing, took, err
}

func (limits *PageVisitLimits) withAnotherDeferral(used usedAllowances) (usedAllowances, bool) {
	if used.deferrals >= limits.maxPerURL.Deferrals {
		return used, false
	}
	used.deferrals++
	return used, true
}

func (limits *PageVisitLimits) TakeAttempt(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (int, bool, error) {
	standing, attempted, err := limits.takeAllowance(ctx, crawlID, url, limits.withAnotherAttempt)
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
