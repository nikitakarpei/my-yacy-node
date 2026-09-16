package jetstream

import "context"

type PeerHistoryObserver interface {
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

type PeerHistoryObservers []PeerHistoryObserver

func (observers PeerHistoryObservers) PeerAnsweredPublishFailed(
	ctx context.Context,
	err error,
) {
	for _, observer := range observers {
		observer.PeerAnsweredPublishFailed(ctx, err)
	}
}

func (observers PeerHistoryObservers) PeerAnsweredMessageUndecodable(
	ctx context.Context,
	sequence uint64,
	err error,
) {
	for _, observer := range observers {
		observer.PeerAnsweredMessageUndecodable(ctx, sequence, err)
	}
}

func (observers PeerHistoryObservers) PeerAnsweredStreamEnded(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.PeerAnsweredStreamEnded(ctx, err)
	}
}

func (observers PeerHistoryObservers) PeerAnsweredStreamPurgeFailed(
	ctx context.Context,
	err error,
) {
	for _, observer := range observers {
		observer.PeerAnsweredStreamPurgeFailed(ctx, err)
	}
}

func (observers PeerHistoryObservers) PeerAnsweredStreamPurged(
	ctx context.Context,
	purgedUpTo uint64,
) {
	for _, observer := range observers {
		observer.PeerAnsweredStreamPurged(ctx, purgedUpTo)
	}
}

func (observers PeerHistoryObservers) SnapshotsReadFailed(ctx context.Context, err error) {
	for _, observer := range observers {
		observer.SnapshotsReadFailed(ctx, err)
	}
}

func (observers PeerHistoryObservers) SnapshotUndecodable(
	ctx context.Context,
	key string,
	err error,
) {
	for _, observer := range observers {
		observer.SnapshotUndecodable(ctx, key, err)
	}
}

func (observers PeerHistoryObservers) SnapshotWriteFailed(
	ctx context.Context,
	key string,
	err error,
) {
	for _, observer := range observers {
		observer.SnapshotWriteFailed(ctx, key, err)
	}
}

func (observers PeerHistoryObservers) PeersSnapshotted(
	ctx context.Context,
	amountOfSnapshots int,
) {
	for _, observer := range observers {
		observer.PeersSnapshotted(ctx, amountOfSnapshots)
	}
}
