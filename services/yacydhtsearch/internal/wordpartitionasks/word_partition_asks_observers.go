package wordpartitionasks

import "context"

type WordPartitionAsksObserver interface {
	WordPartitionAsksPerformed(ctx context.Context, wordPartitionAsks PerformedWordPartitionAsks)
}

type WordPartitionAsksObservers []WordPartitionAsksObserver

func (observers WordPartitionAsksObservers) WordPartitionAsksPerformed(
	ctx context.Context,
	wordPartitionAsks PerformedWordPartitionAsks,
) {
	for _, observer := range observers {
		observer.WordPartitionAsksPerformed(ctx, wordPartitionAsks)
	}
}
