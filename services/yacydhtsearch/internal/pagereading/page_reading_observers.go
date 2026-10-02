package pagereading

import "context"

type PageReadingObserver interface {
	PageReadingPerformed(ctx context.Context, pageReading PerformedPageReading)
	PageReadingRunFinished(ctx context.Context, run FinishedPageReadingRun)
	PagesReadAheadAfterTheFinish(ctx context.Context, amountOfPagesToRead int)
	PagesReadAfterTheFinish(ctx context.Context, amountOfPagesWanted int)
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

func (observers PageReadingObservers) PageReadingRunFinished(
	ctx context.Context,
	run FinishedPageReadingRun,
) {
	for _, observer := range observers {
		observer.PageReadingRunFinished(ctx, run)
	}
}

func (observers PageReadingObservers) PagesReadAheadAfterTheFinish(
	ctx context.Context,
	amountOfPagesToRead int,
) {
	for _, observer := range observers {
		observer.PagesReadAheadAfterTheFinish(ctx, amountOfPagesToRead)
	}
}

func (observers PageReadingObservers) PagesReadAfterTheFinish(
	ctx context.Context,
	amountOfPagesWanted int,
) {
	for _, observer := range observers {
		observer.PagesReadAfterTheFinish(ctx, amountOfPagesWanted)
	}
}
