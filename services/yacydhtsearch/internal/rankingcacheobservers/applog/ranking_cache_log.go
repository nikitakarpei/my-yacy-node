// Package applog reports to the service log how each query was answered.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	msgQueryAnsweredFromCache  = "query answered from a cached ranking"
	msgQueryAnsweredByPeers    = "query answered by peers"
	msgQueryHoldsNoIndexedTerm = "query holds no word long enough to be indexed"
	msgQueryReachedNoPeer      = "query reached no peer, because the directory held none to ask"
)

type RankingCacheLog struct{}

func (RankingCacheLog) QueryAnsweredFromCache(
	ctx context.Context,
	query searchquery.Query,
	amountOfItems int,
) {
	slog.DebugContext(ctx, msgQueryAnsweredFromCache,
		slog.String("query", query.String()),
		slog.Int("items", amountOfItems),
	)
}

func (RankingCacheLog) QueryAnsweredByPeers(
	ctx context.Context,
	query searchquery.Query,
	amountOfItems int,
) {
	slog.DebugContext(ctx, msgQueryAnsweredByPeers,
		slog.String("query", query.String()),
		slog.Int("items", amountOfItems),
	)
}

func (RankingCacheLog) QueryHoldsNoIndexedTerm(ctx context.Context, query searchquery.Query) {
	slog.DebugContext(ctx, msgQueryHoldsNoIndexedTerm, slog.String("query", query.String()))
}

func (RankingCacheLog) QueryReachedNoPeer(ctx context.Context, query searchquery.Query) {
	slog.WarnContext(ctx, msgQueryReachedNoPeer, slog.String("query", query.String()))
}
