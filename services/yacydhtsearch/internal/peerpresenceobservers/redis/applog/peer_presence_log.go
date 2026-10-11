// Package applog reports the snapshots of earned presence to the service log.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgSnapshotsReadFailed = "the snapshots of earned presence could not be read"
	msgSnapshotUndecodable = "a snapshot of earned presence could not be decoded"
	msgSnapshotWriteFailed = "a snapshot of earned presence could not be written"
	msgPeersSnapshotted    = "the earned presence of peers was snapshotted"
)

type PeerPresenceLog struct{}

func (PeerPresenceLog) SnapshotsReadFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgSnapshotsReadFailed, slog.Any("error", err))
}

func (PeerPresenceLog) SnapshotUndecodable(ctx context.Context, key string, err error) {
	slog.WarnContext(ctx, msgSnapshotUndecodable,
		slog.String("key", key),
		slog.Any("error", err),
	)
}

func (PeerPresenceLog) SnapshotWriteFailed(ctx context.Context, key string, err error) {
	slog.WarnContext(ctx, msgSnapshotWriteFailed,
		slog.String("key", key),
		slog.Any("error", err),
	)
}

func (PeerPresenceLog) PeersSnapshotted(ctx context.Context, amountOfSnapshots int) {
	slog.DebugContext(ctx, msgPeersSnapshotted, slog.Int("snapshots", amountOfSnapshots))
}
