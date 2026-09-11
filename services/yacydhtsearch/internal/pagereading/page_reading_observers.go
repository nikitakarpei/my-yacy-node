package pagereading

import "context"

type PageReadingObserver interface {
	PageReadingPerformed(ctx context.Context, pageReading PerformedPageReading)
}

type PageReadingObservers []PageReadingObserver

func (observers PageReadingObservers) PageReadingPerformed(
	ctx context.Context,
	pageReading PerformedPageReading,
) {
	for _, observer := range observers {
		observer.PageReadingPerformed(ctx, pageReading)
	}
}
