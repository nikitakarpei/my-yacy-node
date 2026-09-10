// Package applog reports peer matched search activity to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
)

const msgPeerMatchedSearchPerformed = "peer matched search performed"

type PeerMatchedSearchLog struct{}

func (PeerMatchedSearchLog) PeerMatchedSearchPerformed(
	ctx context.Context,
	search peermatched.PerformedPeerMatchedSearch,
) {
	slog.DebugContext(ctx, msgPeerMatchedSearchPerformed,
		slog.Int("amountOfQueryWords", search.AmountOfQueryWords),
		slog.Int("amountOfAskedPeers", search.AmountOfAskedPeers),
		slog.Int("amountOfPeersThatAnswered", search.AmountOfPeersThatAnswered),
		slog.Int("amountOfPeersThatMatchedNothing", search.AmountOfPeersThatMatchedNothing),
		slog.Duration("timeSpent", search.TimeSpent),
	)
}
