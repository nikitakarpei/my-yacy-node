package jetstream

import "context"

type PeerPresenceObserver interface {
	PeerAnsweredPublishFailed(ctx context.Context, err error)
	PeerAnsweredMessageUndecodable(ctx context.Context, sequence uint64, err error)
	PeerAnsweredStreamEnded(ctx context.Context, err error)
	PeerAnsweredStreamPurgeFailed(ctx context.Context, err error)
	PeerAnsweredStreamPurged(ctx context.Context, purgedUpTo uint64)
	SnapshotsReadFailed(ctx context.Context, err error)
	SnapshotUndecodable(ctx context.Context, key string, err error)
	SnapshotWriteFailed(ctx context.Context, key string, err error)
	PeersSnapshotted(ctx context.Context, amountOfSnapshots int)
}

type PeerPresenceObservers []PeerPresenceObserver

func (observers PeerPresenceObservers) PeerAnsweredPublishFailed(
	ctx context.Context,
	err error,
) {
	for _, observer := range observers {
		observer.PeerAnsweredPublishFailed(ctx, err)
	}
}

func (observers PeerPresenceObservers) PeerAnsweredMessageUndecodable(
	ctx context.Context,
	sequence uint64,
	err error,
) {
	for _, observer := range observers {
		observer.PeerAnsweredMessageUndecodable(ctx, sequence, err)
	}
}

func (observers PeerPresenceObservers) PeerAnsweredStreamEnded(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.PeerAnsweredStreamEnded(ctx, err)
	}
}

func (observers PeerPresenceObservers) PeerAnsweredStreamPurgeFailed(
	ctx context.Context,
	err error,
) {
	for _, observer := range observers {
		observer.PeerAnsweredStreamPurgeFailed(ctx, err)
	}
}

func (observers PeerPresenceObservers) PeerAnsweredStreamPurged(
	ctx context.Context,
	purgedUpTo uint64,
) {
	for _, observer := range observers {
		observer.PeerAnsweredStreamPurged(ctx, purgedUpTo)
	}
}

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
