// Package vault remembers in a vault which message took each URL of a crawl, for the
// retention of taken page visits, so that one message visits the URL however many
// messages carry it.
package vault

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/retainedrecords"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const takersBucket = "taken_page_visits"

var pageVisitKeys = vault.PairKey(vaultkeyparts.CrawlID, vaultkeyparts.URLHash).KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.url },
	func(crawlID crawlid.CrawlID, url urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, url: url}
	},
)

type Clock interface {
	Now() time.Time
}

type TakenPageVisits struct {
	storage *vault.Vault
	takers  *retainedrecords.Records[pageVisitKey, string]
	clock   Clock
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	url     urlhash.URLHash
}

func New(
	ctx context.Context,
	storage *vault.Vault,
	retention *retainedrecords.Retention,
	clock Clock,
	keptFor time.Duration,
) (*TakenPageVisits, error) {
	takers, err := retention.Register(ctx, takersBucket, pageVisitKeys, takerCodec{}, keptFor)
	if err != nil {
		return nil, err
	}
	return &TakenPageVisits{storage: storage, takers: takers, clock: clock}, nil
}

func (visits *TakenPageVisits) TakePageVisit(
	ctx context.Context,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	taker string,
) (bool, error) {
	key := pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)}
	now := visits.clock.Now()
	var took bool
	if err := visits.storage.Update(ctx, func(tx *vault.Txn) error {
		var err error
		took, err = visits.take(tx, key, taker, now)
		return err
	}); err != nil {
		return false, fmt.Errorf("take the page visit to %s: %w", url, err)
	}
	return took, nil
}

func (visits *TakenPageVisits) take(
	tx *vault.Txn,
	key pageVisitKey,
	taker string,
	now time.Time,
) (bool, error) {
	standingTaker, taken, err := visits.takers.Get(tx, key, now)
	if err != nil || taken {
		return standingTaker == taker, err
	}
	return true, visits.takers.Put(tx, key, taker, now)
}

type takerCodec struct{}

func (takerCodec) Encode(taker string) ([]byte, error) { return []byte(taker), nil }

func (takerCodec) Decode(stored []byte) (string, error) { return string(stored), nil }
