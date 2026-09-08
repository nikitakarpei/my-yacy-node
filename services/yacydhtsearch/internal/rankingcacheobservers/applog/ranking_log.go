// Package applog reports to the service log where the ranking cache failed.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	msgRankingLookupFailed = "cached ranking could not be read"
	msgRankingStoreFailed  = "ranking could not be cached"
)

type RankingLog struct{}

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
