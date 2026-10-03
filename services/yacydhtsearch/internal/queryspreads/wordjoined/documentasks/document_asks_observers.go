package documentasks

import "context"

type DocumentAsksObserver interface {
	AskedPartitionFor(ctx context.Context, partition uint, kind Kind)
}

type DocumentAsksObservers []DocumentAsksObserver

func (observers DocumentAsksObservers) AskedPartitionFor(
	ctx context.Context,
	partition uint,
	kind Kind,
) {
	for _, observer := range observers {
		observer.AskedPartitionFor(ctx, partition, kind)
	}
}
