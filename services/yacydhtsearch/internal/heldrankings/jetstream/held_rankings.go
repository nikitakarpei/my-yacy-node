// Package jetstream holds query rankings in a NATS key-value bucket, so that
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

type HeldRankingsObserver interface {
	RankingLookupFailed(ctx context.Context, query searchquery.Query, err error)
	RankingStoreFailed(ctx context.Context, query searchquery.Query, err error)
}

type HeldRankings struct {
	bucket   natsjetstream.KeyValue
	observer HeldRankingsObserver
}

func New(bucket natsjetstream.KeyValue, observer HeldRankingsObserver) *HeldRankings {
	return &HeldRankings{bucket: bucket, observer: observer}
}

func (h *HeldRankings) RankingFor(
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

func (h *HeldRankings) Store(
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

type HeldRankingsObservers []HeldRankingsObserver

func (observers HeldRankingsObservers) RankingLookupFailed(
	ctx context.Context,
	query searchquery.Query,
	err error,
) {
	for _, observer := range observers {
		observer.RankingLookupFailed(ctx, query, err)
	}
}

func (observers HeldRankingsObservers) RankingStoreFailed(
	ctx context.Context,
	query searchquery.Query,
	err error,
) {
	for _, observer := range observers {
		observer.RankingStoreFailed(ctx, query, err)
	}
}
