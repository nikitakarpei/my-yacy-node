package rankingcache

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

type RankingCacheObservers []RankingCacheObserver

func (observers RankingCacheObservers) QueryAnsweredFromCache(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryAnsweredFromCache(ctx, query)
	}
}

func (observers RankingCacheObservers) QueryMissedCache(
	ctx context.Context,
	query searchquery.Query,
) {
	for _, observer := range observers {
		observer.QueryMissedCache(ctx, query)
	}
}
