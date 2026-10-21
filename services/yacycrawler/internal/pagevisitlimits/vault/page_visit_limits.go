// Package vault holds in a vault what a crawl has already used for one page visit
// — a page of the URL's host, the deferrals of that URL and its attempts — and
// admits another only while the crawl's limit leaves room for it. A redelivered
// page visit takes the page of its host once.
package vault

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/retainedrecords"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

var errLimitMalformed = errors.New("page visit limit record is malformed")

const (
	takenHostPageCountsBucket = "taken_host_page_counts"
	usedAllowancesBucket      = "used_page_visit_allowances"
)

var hostKeys = vault.PairKey(vaultkeyparts.CrawlID, vault.TextKeyPart).KeyLayoutFor(
	func(key hostKey) (crawlid.CrawlID, string) { return key.crawlID, key.host },
	func(crawlID crawlid.CrawlID, host string) hostKey {
		return hostKey{crawlID: crawlID, host: host}
	},
)

var pageVisitKeys = vault.PairKey(vaultkeyparts.CrawlID, vaultkeyparts.URLHash).KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.url },
	func(crawlID crawlid.CrawlID, url urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, url: url}
	},
)

type MaxPerURL struct {
	Deferrals int
	Attempts  int
}

type Clock interface {
	Now() time.Time
}

type PageVisitLimits struct {
	storage             *vault.Vault
	takenHostPageCounts *retainedrecords.Records[hostKey, takenHostPageCount]
	usedAllowances      *retainedrecords.Records[pageVisitKey, usedAllowances]
	clock               Clock
	maxPerURL           MaxPerURL
}

type hostKey struct {
	crawlID crawlid.CrawlID
	host    string
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	url     urlhash.URLHash
}

//nolint:revive // the limits name the vault, its retention and the ceilings they admit within
func New(
	ctx context.Context,
	storage *vault.Vault,
	retention *retainedrecords.Retention,
	clock Clock,
	keptFor time.Duration,
	maxPerURL MaxPerURL,
) (*PageVisitLimits, error) {
	takenHostPageCountsKept, err := retention.Register(
		ctx, takenHostPageCountsBucket, hostKeys, takenHostPageCountCodec{}, keptFor,
	)
	if err != nil {
		return nil, err
	}
	usedAllowancesKept, err := retention.Register(
		ctx, usedAllowancesBucket, pageVisitKeys, usedAllowancesCodec{}, keptFor,
	)
	if err != nil {
		return nil, err
	}
	return &PageVisitLimits{
		storage:             storage,
		takenHostPageCounts: takenHostPageCountsKept,
		usedAllowances:      usedAllowancesKept,
		clock:               clock,
		maxPerURL:           maxPerURL,
	}, nil
}

func (limits *PageVisitLimits) TakePageOfHost(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	host string,
	maxPages int,
) (bool, error) {
	if maxPages == yacycrawlcontract.UnlimitedPagesPerHost {
		return true, nil
	}
	pageVisit := pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)}
	crawlHost := hostKey{crawlID: crawlID, host: host}
	now := limits.clock.Now()
	var took bool
	if err := limits.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		took, err = limits.takePageOfHost(tx, pageVisit, crawlHost, maxPages, now)
		return err
	}); err != nil {
		return false, fmt.Errorf("take a page of host %s for %s: %w", host, url, err)
	}
	return took, nil
}

func (limits *PageVisitLimits) takePageOfHost(
	tx *vault.Txn,
	pageVisit pageVisitKey,
	crawlHost hostKey,
	maxPages int,
	now time.Time,
) (bool, error) {
	allowances, _, err := limits.usedAllowances.Get(tx, pageVisit, now)
	if err != nil || allowances.holdsPageOfHost {
		return allowances.holdsPageOfHost, err
	}
	pageCount, _, err := limits.takenHostPageCounts.Get(tx, crawlHost, now)
	if err != nil || pageCount.pages >= maxPages {
		return false, err
	}
	pageCount.pages++
	if err := limits.takenHostPageCounts.Put(tx, crawlHost, pageCount, now); err != nil {
		return false, err
	}
	allowances.holdsPageOfHost = true
	return true, limits.usedAllowances.Put(tx, pageVisit, allowances, now)
}

func (limits *PageVisitLimits) TakeDeferral(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	_, deferred, err := limits.revisePageVisit(ctx, crawlID, url, limits.withAnotherDeferral)
	if err != nil {
		return false, fmt.Errorf("take a deferral of the page visit to %s: %w", url, err)
	}
	return deferred, nil
}

func (limits *PageVisitLimits) revisePageVisit(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	revise func(usedAllowances) (usedAllowances, bool),
) (usedAllowances, bool, error) {
	key := pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)}
	now := limits.clock.Now()
	var standing usedAllowances
	var revised bool
	err := limits.storage.Update(ctx, func(tx *vault.Txn) error {
		allowances, _, err := limits.usedAllowances.Get(tx, key, now)
		if err != nil {
			return err
		}
		standing, revised = revise(allowances)
		if !revised {
			return nil
		}
		return limits.usedAllowances.Put(tx, key, standing, now)
	})
	return standing, revised, err
}

func (limits *PageVisitLimits) withAnotherDeferral(
	allowances usedAllowances,
) (usedAllowances, bool) {
	if allowances.deferrals >= limits.maxPerURL.Deferrals {
		return allowances, false
	}
	allowances.deferrals++
	return allowances, true
}

func (limits *PageVisitLimits) TakeAttempt(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (int, bool, error) {
	standing, attempted, err := limits.revisePageVisit(ctx, crawlID, url, limits.withAnotherAttempt)
	if err != nil {
		return 0, false, fmt.Errorf("take an attempt of the page visit to %s: %w", url, err)
	}
	return standing.attempts, attempted, nil
}

func (limits *PageVisitLimits) withAnotherAttempt(
	allowances usedAllowances,
) (usedAllowances, bool) {
	if allowances.attempts >= limits.maxPerURL.Attempts {
		return allowances, false
	}
	allowances.attempts++
	return allowances, true
}

type takenHostPageCount struct {
	pages int
}

type takenHostPageCountCodec struct{}

func (takenHostPageCountCodec) Encode(record takenHostPageCount) ([]byte, error) {
	return binary.AppendVarint(nil, int64(record.pages)), nil
}

func (takenHostPageCountCodec) Decode(stored []byte) (takenHostPageCount, error) {
	pages, pagesBytes := binary.Varint(stored)
	if pagesBytes != len(stored) {
		return takenHostPageCount{}, errLimitMalformed
	}
	return takenHostPageCount{pages: int(pages)}, nil
}

type usedAllowances struct {
	holdsPageOfHost bool
	deferrals       int
	attempts        int
}

type usedAllowancesCodec struct{}

func (usedAllowancesCodec) Encode(record usedAllowances) ([]byte, error) {
	encoded := make([]byte, 0, 1+2*binary.MaxVarintLen64)
	if record.holdsPageOfHost {
		encoded = append(encoded, 1)
	} else {
		encoded = append(encoded, 0)
	}
	encoded = binary.AppendVarint(encoded, int64(record.deferrals))
	return binary.AppendVarint(encoded, int64(record.attempts)), nil
}

func (usedAllowancesCodec) Decode(stored []byte) (usedAllowances, error) {
	if len(stored) == 0 {
		return usedAllowances{}, errLimitMalformed
	}
	deferrals, deferralsBytes := binary.Varint(stored[1:])
	if deferralsBytes <= 0 {
		return usedAllowances{}, errLimitMalformed
	}
	attempts, attemptsBytes := binary.Varint(stored[1+deferralsBytes:])
	if attemptsBytes <= 0 {
		return usedAllowances{}, errLimitMalformed
	}
	return usedAllowances{
		holdsPageOfHost: stored[0] == 1,
		deferrals:       int(deferrals),
		attempts:        int(attempts),
	}, nil
}
