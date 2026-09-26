// Package applog reports to the service log whether the ranking cache answered
// each query.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	msgQueryAnsweredFromCache = "query answered from a cached ranking"
	msgQueryMissedCache       = "query found no cached ranking"
)

type RankingCacheLog struct{}

func (RankingCacheLog) QueryAnsweredFromCache(ctx context.Context, query searchquery.Query) {
	slog.DebugContext(ctx, msgQueryAnsweredFromCache, slog.String("query", query.String()))
}

func (RankingCacheLog) QueryMissedCache(ctx context.Context, query searchquery.Query) {
	slog.DebugContext(ctx, msgQueryMissedCache, slog.String("query", query.String()))
}
