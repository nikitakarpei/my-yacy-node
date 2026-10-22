package takenpagevisits

import (
	"context"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const takersBucket = "taken_page_visits"

func registerTakenPageVisits(
	ctx context.Context,
	storage *vault.Vault,
	keptFor time.Duration,
) (*vault.ExpiringCollection[pageVisitKey, string], error) {
	takers, err := storage.RegisterExpiringCollection(
		ctx, takersBucket, pageVisitKeys, takerCodec{}, keptFor, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("register %s: %w", takersBucket, err)
	}
	return takers, nil
}

var pageVisitKeys = vault.PairKey(vaultkeyparts.CrawlID, vaultkeyparts.URLHash).KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.url },
	func(crawlID crawlid.CrawlID, url urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, url: url}
	},
)

type takerCodec struct{}

func (takerCodec) Encode(taker string) ([]byte, error) { return []byte(taker), nil }

func (takerCodec) Decode(stored []byte) (string, error) { return string(stored), nil }
