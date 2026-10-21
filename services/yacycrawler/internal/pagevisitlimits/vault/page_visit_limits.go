// Package vault holds in a vault what a crawl has already taken for one page visit
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
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlkeyparts"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/retainedrecords"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

var errLimitMalformed = errors.New("page visit limit record is malformed")

const (
	takenHostPageCountsBucket = "taken_host_page_counts"
	pageVisitsBucket          = "taken_for_page_visits"
)

var hostKeys = vault.PairKey(crawlkeyparts.CrawlID, vault.TextKeyPart).KeyLayoutFor(
	func(key hostKey) (crawlid.CrawlID, string) { return key.crawlID, key.host },
	func(crawlID crawlid.CrawlID, host string) hostKey {
		return hostKey{crawlID: crawlID, host: host}
	},
)

var pageVisitKeys = vault.PairKey(crawlkeyparts.CrawlID, crawlkeyparts.URLHash).KeyLayoutFor(
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
	pageVisits          *retainedrecords.Records[pageVisitKey, pageVisitTaken]
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
	pageVisitsKept, err := retention.Register(
		ctx, pageVisitsBucket, pageVisitKeys, pageVisitTakenCodec{}, keptFor,
	)
	if err != nil {
		return nil, err
	}
	return &PageVisitLimits{
		storage:             storage,
		takenHostPageCounts: takenHostPageCountsKept,
		pageVisits:          pageVisitsKept,
		clock:               clock,
		maxPerURL:           maxPerURL,
	}, nil
}

func (limits *PageVisitLimits) AdmitsAnotherPageOfHost(
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
	pagesOfHost := hostKey{crawlID: crawlID, host: host}
	now := limits.clock.Now()
	var admitted bool
	if err := limits.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		admitted, err = limits.holdsPageOfHost(tx, pageVisit, pagesOfHost, maxPages, now)
		return err
	}); err != nil {
		return false, fmt.Errorf("take a page of host %s for %s: %w", host, url, err)
	}
	return admitted, nil
}

func (limits *PageVisitLimits) holdsPageOfHost(
	tx *vault.Txn,
	pageVisit pageVisitKey,
	pagesOfHost hostKey,
	maxPages int,
	now time.Time,
) (bool, error) {
	taken, _, err := limits.pageVisits.Get(tx, pageVisit, now)
	if err != nil || taken.holdsPageOfHost {
		return taken.holdsPageOfHost, err
	}
	pages, _, err := limits.takenHostPageCounts.Get(tx, pagesOfHost, now)
	if err != nil || pages.pages >= maxPages {
		return false, err
	}
	pages.pages++
	if err := limits.takenHostPageCounts.Put(tx, pagesOfHost, pages, now); err != nil {
		return false, err
	}
	taken.holdsPageOfHost = true
	return true, limits.pageVisits.Put(tx, pageVisit, taken, now)
}

func (limits *PageVisitLimits) AdmitsAnotherDeferral(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (bool, error) {
	_, deferred, err := limits.revisePageVisit(ctx, crawlID, url, limits.anotherDeferral)
	if err != nil {
		return false, fmt.Errorf("take a deferral of the page visit to %s: %w", url, err)
	}
	return deferred, nil
}

func (limits *PageVisitLimits) revisePageVisit(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	revise func(pageVisitTaken) (pageVisitTaken, bool),
) (pageVisitTaken, bool, error) {
	key := pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)}
	now := limits.clock.Now()
	var standing pageVisitTaken
	var revised bool
	err := limits.storage.Update(ctx, func(tx *vault.Txn) error {
		taken, _, err := limits.pageVisits.Get(tx, key, now)
		if err != nil {
			return err
		}
		standing, revised = revise(taken)
		if !revised {
			return nil
		}
		return limits.pageVisits.Put(tx, key, standing, now)
	})
	return standing, revised, err
}

func (limits *PageVisitLimits) anotherDeferral(taken pageVisitTaken) (pageVisitTaken, bool) {
	if taken.deferrals >= limits.maxPerURL.Deferrals {
		return taken, false
	}
	taken.deferrals++
	return taken, true
}

func (limits *PageVisitLimits) AdmitsAnotherAttempt(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
) (int, bool, error) {
	standing, attempted, err := limits.revisePageVisit(ctx, crawlID, url, limits.anotherAttempt)
	if err != nil {
		return 0, false, fmt.Errorf("take an attempt of the page visit to %s: %w", url, err)
	}
	return standing.attempts, attempted, nil
}

func (limits *PageVisitLimits) anotherAttempt(taken pageVisitTaken) (pageVisitTaken, bool) {
	if taken.attempts >= limits.maxPerURL.Attempts {
		return taken, false
	}
	taken.attempts++
	return taken, true
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

type pageVisitTaken struct {
	holdsPageOfHost bool
	deferrals       int
	attempts        int
}

type pageVisitTakenCodec struct{}

func (pageVisitTakenCodec) Encode(record pageVisitTaken) ([]byte, error) {
	encoded := make([]byte, 0, 1+2*binary.MaxVarintLen64)
	if record.holdsPageOfHost {
		encoded = append(encoded, 1)
	} else {
		encoded = append(encoded, 0)
	}
	encoded = binary.AppendVarint(encoded, int64(record.deferrals))
	return binary.AppendVarint(encoded, int64(record.attempts)), nil
}

func (pageVisitTakenCodec) Decode(stored []byte) (pageVisitTaken, error) {
	if len(stored) == 0 {
		return pageVisitTaken{}, errLimitMalformed
	}
	deferrals, deferralsBytes := binary.Varint(stored[1:])
	if deferralsBytes <= 0 {
		return pageVisitTaken{}, errLimitMalformed
	}
	attempts, attemptsBytes := binary.Varint(stored[1+deferralsBytes:])
	if attemptsBytes <= 0 {
		return pageVisitTaken{}, errLimitMalformed
	}
	return pageVisitTaken{
		holdsPageOfHost: stored[0] == 1,
		deferrals:       int(deferrals),
		attempts:        int(attempts),
	}, nil
}
