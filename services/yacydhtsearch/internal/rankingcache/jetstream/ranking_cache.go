// Package jetstream caches query rankings in a NATS key-value bucket, so that
// every service instance answers a repeated query from the same ranking.
package jetstream

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
)

type RankingCacheObserver interface {
	RankingLookupFailed(ctx context.Context, query searchquery.Query, err error)
	RankingStoreFailed(ctx context.Context, query searchquery.Query, err error)
}

type RankingCache struct {
	bucket   natsjetstream.KeyValue
	observer RankingCacheObserver
}

func New(bucket natsjetstream.KeyValue, observer RankingCacheObserver) *RankingCache {
	return &RankingCache{bucket: bucket, observer: observer}
}

func (h *RankingCache) CachedRankingFor(
	ctx context.Context,
	query searchquery.Query,
) (searchresult.Ranking, bool) {
	entry, err := h.bucket.Get(ctx, keyFor(query))
	if errors.Is(err, natsjetstream.ErrKeyNotFound) {
		return searchresult.Ranking{}, false
	}
	if err != nil {
		h.observer.RankingLookupFailed(ctx, query, err)

		return searchresult.Ranking{}, false
	}

	var ranking searchresult.Ranking
	if err := json.Unmarshal(entry.Value(), &ranking); err != nil {
		h.observer.RankingLookupFailed(ctx, query, err)

		return searchresult.Ranking{}, false
	}

	return ranking, true
}

func (h *RankingCache) StoreRanking(
	ctx context.Context,
	query searchquery.Query,
	ranking searchresult.Ranking,
) {
	encoded, err := json.Marshal(ranking)
	if err != nil {
		h.observer.RankingStoreFailed(ctx, query, err)

		return
	}
	if _, err := h.bucket.Put(ctx, keyFor(query), encoded); err != nil {
		h.observer.RankingStoreFailed(ctx, query, err)
	}
}

func keyFor(query searchquery.Query) string {
	spelled := sha256.Sum256([]byte(query.String()))

	return base64.RawURLEncoding.EncodeToString(spelled[:])
}

type RankingCacheObservers []RankingCacheObserver

func (observers RankingCacheObservers) RankingLookupFailed(
	ctx context.Context,
	query searchquery.Query,
	err error,
) {
	for _, observer := range observers {
		observer.RankingLookupFailed(ctx, query, err)
	}
}

func (observers RankingCacheObservers) RankingStoreFailed(
	ctx context.Context,
	query searchquery.Query,
	err error,
) {
	for _, observer := range observers {
		observer.RankingStoreFailed(ctx, query, err)
	}
}
