package urlmetadataaskceilings

import "context"

type AskCeilingObserver interface {
	AskCeilingSet(ctx context.Context, address string, documentsPerSecond float64, ceiling int)
	AskCeilingHandedOut(ctx context.Context, address string, ceiling int)
}

type AskCeilingObservers []AskCeilingObserver

func (observers AskCeilingObservers) AskCeilingSet(
	ctx context.Context,
	address string,
	documentsPerSecond float64,
	ceiling int,
) {
	for _, observer := range observers {
		observer.AskCeilingSet(ctx, address, documentsPerSecond, ceiling)
	}
}

func (observers AskCeilingObservers) AskCeilingHandedOut(
	ctx context.Context,
	address string,
	ceiling int,
) {
	for _, observer := range observers {
		observer.AskCeilingHandedOut(ctx, address, ceiling)
	}
}
