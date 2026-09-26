// Package applog reports to the service log where holding rankings in NATS failed.
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

type HeldRankingsLog struct{}

func (HeldRankingsLog) RankingLookupFailed(
	ctx context.Context,
	query searchquery.Query,
	err error,
) {
	slog.WarnContext(ctx, msgRankingLookupFailed,
		slog.String("query", query.String()),
		slog.Any("error", err),
	)
}

func (HeldRankingsLog) RankingStoreFailed(ctx context.Context, query searchquery.Query, err error) {
	slog.WarnContext(ctx, msgRankingStoreFailed,
		slog.String("query", query.String()),
		slog.Any("error", err),
	)
}
