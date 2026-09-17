// Package applog reports the presence peers earn to the service log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	msgPeerAnsweredForTheFirstTime = "peer answered for the first time"
	msgPeerEarnedPresence          = "peer earned presence"
)

type PresenceAccrualLog struct{}

func (PresenceAccrualLog) PeerAnsweredForTheFirstTime(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
) {
	slog.DebugContext(ctx, msgPeerAnsweredForTheFirstTime,
		slog.String("peer", peer.String()),
		slog.String("address", address),
	)
}

func (PresenceAccrualLog) PeerEarnedPresence(
	ctx context.Context,
	peer yacymodel.Hash,
	presence time.Duration,
) {
	slog.DebugContext(ctx, msgPeerEarnedPresence,
		slog.String("peer", peer.String()),
		slog.Duration("presence", presence),
	)
}
