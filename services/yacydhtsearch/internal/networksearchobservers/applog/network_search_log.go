// Package applog reports what one network search did to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
)

const msgNetworkSearchPerformed = "network search performed"

type NetworkSearchLog struct{}

func (NetworkSearchLog) NetworkSearchPerformed(
	ctx context.Context,
	search networksearch.PerformedNetworkSearch,
) {
	slog.DebugContext(ctx, msgNetworkSearchPerformed,
		slog.Int("amountOfAskablePeers", search.AmountOfAskablePeers),
		slog.Int("amountOfItemsAcrossAnswers", search.AmountOfItemsAcrossAnswers),
		slog.Int("amountOfRankedItemsOfTheOnePeer", search.AmountOfRankedItemsOfTheOnePeer),
		slog.Int("amountOfItemsInRanking", search.AmountOfItemsInRanking),
		slog.Int("amountOfRankedItemsCountedByAPeer", search.AmountOfRankedItemsCountedByAPeer),
		slog.Duration("timeSpent", search.TimeSpent),
	)
}
