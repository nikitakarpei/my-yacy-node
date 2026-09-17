// Package applog reports peer directory changes to the service log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	msgPeerAdmitted   = "peer admitted to the directory"
	msgPeerAnswered   = "peer answered on an address"
	msgPeerWentSilent = "peer went silent on every address"
	msgPeerDropped    = "peer dropped from the directory"
	msgPeersKnown     = "peer directory holds peers"
)

type DirectoryLog struct{}

func (DirectoryLog) PeerAdmitted(ctx context.Context, peer yacymodel.Hash, addresses int) {
	slog.DebugContext(ctx, msgPeerAdmitted,
		slog.String("peer", peer.String()),
		slog.Int("addresses", addresses),
	)
}

func (DirectoryLog) PeerAnswered(
	ctx context.Context,
	peer yacymodel.Hash,
	address string,
	answeredAt time.Time,
) {
	slog.DebugContext(ctx, msgPeerAnswered,
		slog.String("peer", peer.String()),
		slog.String("address", address),
		slog.Time("answeredAt", answeredAt),
	)
}

func (DirectoryLog) PeerWentSilent(ctx context.Context, peer yacymodel.Hash) {
	slog.DebugContext(ctx, msgPeerWentSilent, slog.String("peer", peer.String()))
}

func (DirectoryLog) PeerDropped(ctx context.Context, peer yacymodel.Hash) {
	slog.DebugContext(ctx, msgPeerDropped, slog.String("peer", peer.String()))
}

func (DirectoryLog) PeersKnown(
	ctx context.Context,
	amountOfPeers, amountOfAnsweringPeers, capacity int,
) {
	slog.DebugContext(ctx, msgPeersKnown,
		slog.Int("peers", amountOfPeers),
		slog.Int("answeringPeers", amountOfAnsweringPeers),
		slog.Int("capacity", capacity),
	)
}
