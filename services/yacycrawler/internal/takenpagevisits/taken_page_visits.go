// Package takenpagevisits remembers which message took each URL of a crawl, for the
// retention of taken page visits, so that one message visits the URL however many
// messages carry it.
package takenpagevisits

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
)

type TakenPageVisits struct {
	takers *vault.ExpiringCollection[pageVisitKey, string]
}

func Open(
	ctx context.Context,
	storage *vault.Vault,
	keptFor time.Duration,
) (*TakenPageVisits, error) {
	takers, err := registerTakenPageVisits(ctx, storage, keptFor)
	if err != nil {
		return nil, err
	}
	return &TakenPageVisits{takers: takers}, nil
}

type pageVisitKey struct {
	crawlID crawlid.CrawlID
	url     urlhash.URLHash
}

func (visits *TakenPageVisits) TakePageVisit(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	url canonicalurl.CanonicalURL,
	taker string,
) (bool, error) {
	key := pageVisitKey{crawlID: crawlID, url: urlhash.Of(url)}
	standingTaker, taken, err := visits.takers.Get(tx, key)
	if err != nil {
		return false, fmt.Errorf("take the page visit to %s: %w", url, err)
	}
	if taken {
		return standingTaker == taker, nil
	}
	if err := visits.takers.Put(tx, key, taker); err != nil {
		return false, fmt.Errorf("take the page visit to %s: %w", url, err)
	}
	return true, nil
}
