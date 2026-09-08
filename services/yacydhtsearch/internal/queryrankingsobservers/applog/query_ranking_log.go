// Package applog reports to the service log how each query was answered.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	msgQueryAnsweredFromCache = "query answered from a cached ranking"
	msgQueryAnsweredByPeers   = "query answered by peers"
	msgQueryHadNoIndexedTerm  = "query holds no word long enough to be indexed"
	msgQueryFoundNoPeerToAsk  = "query reached no peer, because the directory held none to ask"
)

type QueryRankingLog struct{}

func (QueryRankingLog) QueryAnsweredFromCache(
	ctx context.Context,
	query searchquery.Query,
	items int,
) {
	slog.DebugContext(ctx, msgQueryAnsweredFromCache,
		slog.String("query", query.String()),
		slog.Int("items", items),
	)
}

func (QueryRankingLog) QueryAnsweredByPeers(
	ctx context.Context,
	query searchquery.Query,
	items int,
) {
	slog.DebugContext(ctx, msgQueryAnsweredByPeers,
		slog.String("query", query.String()),
		slog.Int("items", items),
	)
}

func (QueryRankingLog) QueryHadNoIndexedTerm(ctx context.Context, query searchquery.Query) {
	slog.DebugContext(ctx, msgQueryHadNoIndexedTerm, slog.String("query", query.String()))
}

func (QueryRankingLog) QueryFoundNoPeerToAsk(ctx context.Context, query searchquery.Query) {
	slog.WarnContext(ctx, msgQueryFoundNoPeerToAsk, slog.String("query", query.String()))
}
