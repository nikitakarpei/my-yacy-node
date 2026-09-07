// Package applog reports peer directory changes to the service log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	msgPeerAdmitted   = "peer admitted to the directory"
	msgPeerAnswering  = "peer answers on an address"
	msgPeerSilent     = "peer answered on no address"
	msgPeerDropped    = "peer dropped from the directory"
	msgDirectoryHolds = "peer directory holds peers"
)

type DirectoryLog struct{}

func (DirectoryLog) PeerAdmitted(ctx context.Context, peer yacymodel.Hash, addresses int) {
	slog.DebugContext(ctx, msgPeerAdmitted,
		slog.String("peer", peer.String()),
		slog.Int("addresses", addresses),
	)
}

func (DirectoryLog) PeerAnswering(ctx context.Context, peer yacymodel.Hash, address string) {
	slog.DebugContext(ctx, msgPeerAnswering,
		slog.String("peer", peer.String()),
		slog.String("address", address),
	)
}

func (DirectoryLog) PeerSilent(ctx context.Context, peer yacymodel.Hash) {
	slog.DebugContext(ctx, msgPeerSilent, slog.String("peer", peer.String()))
}

func (DirectoryLog) PeerDropped(ctx context.Context, peer yacymodel.Hash) {
	slog.DebugContext(ctx, msgPeerDropped, slog.String("peer", peer.String()))
}

func (DirectoryLog) DirectoryHolds(ctx context.Context, peers, answeringPeers, capacity int) {
	slog.DebugContext(ctx, msgDirectoryHolds,
		slog.Int("peers", peers),
		slog.Int("answeringPeers", answeringPeers),
		slog.Int("capacity", capacity),
	)
}
