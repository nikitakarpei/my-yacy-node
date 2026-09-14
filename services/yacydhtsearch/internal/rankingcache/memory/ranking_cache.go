// Package memory caches query rankings in this process, for as long as their
// lifetime allows and as many as its capacity allows.
package memory

import (
	"context"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type RankingCache struct {
	rankings *expirable.LRU[string, searchresult.Ranking]
}

func New(capacity int, lifetime time.Duration) *RankingCache {
	return &RankingCache{
		rankings: expirable.NewLRU[string, searchresult.Ranking](capacity, nil, lifetime),
	}
}

func (h *RankingCache) CachedRankingFor(
	_ context.Context,
	query searchquery.Query,
) (searchresult.Ranking, bool) {
	return h.rankings.Get(query.String())
}

func (h *RankingCache) StoreRanking(
	_ context.Context,
	query searchquery.Query,
	ranking searchresult.Ranking,
) {
	h.rankings.Add(query.String(), ranking)
}
