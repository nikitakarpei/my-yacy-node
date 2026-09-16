package jetstream

import "context"

type PeerPresenceObserver interface {
	SnapshotsReadFailed(ctx context.Context, err error)
	SnapshotUndecodable(ctx context.Context, key string, err error)
	SnapshotWriteFailed(ctx context.Context, key string, err error)
	PeersSnapshotted(ctx context.Context, amountOfSnapshots int)
}

type PeerPresenceObservers []PeerPresenceObserver

func (observers PeerPresenceObservers) SnapshotsReadFailed(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.SnapshotsReadFailed(ctx, err)
	}
}

func (observers PeerPresenceObservers) SnapshotUndecodable(
	ctx context.Context,
	key string,
	err error,
) {
	for _, observer := range observers {
		observer.SnapshotUndecodable(ctx, key, err)
	}
}

func (observers PeerPresenceObservers) SnapshotWriteFailed(
	ctx context.Context,
	key string,
	err error,
) {
	for _, observer := range observers {
		observer.SnapshotWriteFailed(ctx, key, err)
	}
}

func (observers PeerPresenceObservers) PeersSnapshotted(
	ctx context.Context,
	amountOfSnapshots int,
) {
	for _, observer := range observers {
		observer.PeersSnapshotted(ctx, amountOfSnapshots)
	}
}
