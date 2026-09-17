package replicaasks

import "context"

type ReplicaAsksObserver interface {
	ReplicaAsksPerformed(ctx context.Context, replicaAsks PerformedReplicaAsks)
}

type ReplicaAsksObservers []ReplicaAsksObserver

func (observers ReplicaAsksObservers) ReplicaAsksPerformed(
	ctx context.Context,
	replicaAsks PerformedReplicaAsks,
) {
	for _, observer := range observers {
		observer.ReplicaAsksPerformed(ctx, replicaAsks)
	}
}
