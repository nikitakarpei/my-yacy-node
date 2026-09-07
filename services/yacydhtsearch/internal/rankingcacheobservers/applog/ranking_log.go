// Package applog reports to the service log which queries a cached ranking
// answered and where the ranking cache failed.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	msgCachedRankingAnswered  = "query answered from a cached ranking"
	msgNetworkRankingAnswered = "query answered from the network"
	msgRankingLookupFailed    = "cached ranking could not be read"
	msgRankingStoreFailed     = "ranking could not be cached"
)

type RankingLog struct{}

func (RankingLog) CachedRankingAnswered(ctx context.Context, query searchquery.Query, items int) {
	slog.DebugContext(ctx, msgCachedRankingAnswered,
		slog.String("query", query.String()),
		slog.Int("items", items),
	)
}

func (RankingLog) NetworkRankingAnswered(ctx context.Context, query searchquery.Query, items int) {
	slog.DebugContext(ctx, msgNetworkRankingAnswered,
		slog.String("query", query.String()),
		slog.Int("items", items),
	)
}

func (RankingLog) RankingLookupFailed(
	ctx context.Context,
	query searchquery.Query,
	err error,
) {
	slog.WarnContext(ctx, msgRankingLookupFailed,
		slog.String("query", query.String()),
		slog.Any("error", err),
	)
}

func (RankingLog) RankingStoreFailed(ctx context.Context, query searchquery.Query, err error) {
	slog.WarnContext(ctx, msgRankingStoreFailed,
		slog.String("query", query.String()),
		slog.Any("error", err),
	)
}
