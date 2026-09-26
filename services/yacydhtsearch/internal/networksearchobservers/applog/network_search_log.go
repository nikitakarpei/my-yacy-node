// Package applog reports what one network search did to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	msgNetworkSearchPerformed = "network search performed"
	msgQueryReachedNoPeer     = "query reached no peer, because the directory held none to ask"
)

type NetworkSearchLog struct{}

func (NetworkSearchLog) NetworkSearchPerformed(
	ctx context.Context,
	search networksearch.PerformedNetworkSearch,
) {
	slog.DebugContext(
		ctx,
		msgNetworkSearchPerformed,
		slog.Int("amountOfAskablePeers", search.AmountOfAskablePeers),
		slog.Int("amountOfFoundDocuments", search.AmountOfFoundDocuments),
		slog.Int("amountOfItemsInRanking", search.AmountOfItemsInRanking),
		slog.Duration("timeSpent", search.TimeSpent),
	)
}

func (NetworkSearchLog) QueryReachedNoPeer(ctx context.Context, query searchquery.Query) {
	slog.WarnContext(ctx, msgQueryReachedNoPeer, slog.String("query", query.String()))
}
