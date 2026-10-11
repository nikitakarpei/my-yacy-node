// Package redis caches query rankings on a Redis-protocol server for the lifetime of a
// ranking, so that every service instance answers a repeated query from the same ranking.
package redis

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type CachedRankingsObserver interface {
	RankingLookupFailed(ctx context.Context, query searchquery.Query, err error)
	RankingStoreFailed(ctx context.Context, query searchquery.Query, err error)
}

const keyPrefix = "rankings:"

type CachedRankings struct {
	client   redis.UniversalClient
	lifetime time.Duration
	observer CachedRankingsObserver
}

func New(
	client redis.UniversalClient,
	lifetime time.Duration,
	observer CachedRankingsObserver,
) *CachedRankings {
	return &CachedRankings{client: client, lifetime: lifetime, observer: observer}
}

func (h *CachedRankings) RankingFor(
	ctx context.Context,
	query searchquery.Query,
) (searchresult.Ranking, bool) {
	encoded, err := h.client.Get(ctx, keyFor(query)).Bytes()
	if errors.Is(err, redis.Nil) {
		return searchresult.Ranking{}, false
	}
	if err != nil {
		h.observer.RankingLookupFailed(ctx, query, err)

		return searchresult.Ranking{}, false
	}

	var ranking searchresult.Ranking
	if err := json.Unmarshal(encoded, &ranking); err != nil {
		h.observer.RankingLookupFailed(ctx, query, err)

		return searchresult.Ranking{}, false
	}

	return ranking, true
}

func (h *CachedRankings) Store(
	ctx context.Context,
	query searchquery.Query,
	ranking searchresult.Ranking,
) {
	encoded, err := json.Marshal(ranking)
	if err != nil {
		h.observer.RankingStoreFailed(ctx, query, err)

		return
	}
	if err := h.client.Set(ctx, keyFor(query), encoded, h.lifetime).Err(); err != nil {
		h.observer.RankingStoreFailed(ctx, query, err)
	}
}

func keyFor(query searchquery.Query) string {
	spelled := sha256.Sum256([]byte(query.String()))

	return keyPrefix + base64.RawURLEncoding.EncodeToString(spelled[:])
}

type CachedRankingsObservers []CachedRankingsObserver

func (observers CachedRankingsObservers) RankingLookupFailed(
	ctx context.Context,
	query searchquery.Query,
	err error,
) {
	for _, observer := range observers {
		observer.RankingLookupFailed(ctx, query, err)
	}
}

func (observers CachedRankingsObservers) RankingStoreFailed(
	ctx context.Context,
	query searchquery.Query,
	err error,
) {
	for _, observer := range observers {
		observer.RankingStoreFailed(ctx, query, err)
	}
}
