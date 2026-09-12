// Package applog reports peer matched spread activity to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
)

const msgPeerMatchedSpreadPerformed = "peer matched spread performed"

type PeerMatchedSpreadLog struct{}

func (PeerMatchedSpreadLog) PeerMatchedSpreadPerformed(
	ctx context.Context,
	spread peermatched.PerformedPeerMatchedSpread,
) {
	slog.DebugContext(ctx, msgPeerMatchedSpreadPerformed,
		slog.Int("amountOfQueryWords", spread.AmountOfQueryWords),
		slog.Int("amountOfPeersAsked", spread.AmountOfPeersAsked),
		slog.Int("amountOfPeersThatAnswered", spread.AmountOfPeersThatAnswered),
		slog.Int("amountOfPeersThatMatchedNothing", spread.AmountOfPeersThatMatchedNothing),
		slog.Duration("timeSpent", spread.TimeSpent),
	)
}
