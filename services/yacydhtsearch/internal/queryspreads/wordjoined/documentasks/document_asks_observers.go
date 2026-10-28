package documentasks

import "context"

type DocumentAsksObserver interface {
	AskedAmongTheDocuments(
		ctx context.Context,
		decisionPerPartition DocumentsToMatchDecisionPerPartition,
	)
	ListingsEnded(ctx context.Context, ended EndedListings)
	DocumentAsksPerformed(ctx context.Context, performed Performed)
}

type DocumentAsksObservers []DocumentAsksObserver

func (observers DocumentAsksObservers) AskedAmongTheDocuments(
	ctx context.Context,
	decisionPerPartition DocumentsToMatchDecisionPerPartition,
) {
	for _, observer := range observers {
		observer.AskedAmongTheDocuments(ctx, decisionPerPartition)
	}
}

func (observers DocumentAsksObservers) ListingsEnded(ctx context.Context, ended EndedListings) {
	for _, observer := range observers {
		observer.ListingsEnded(ctx, ended)
	}
}

func (observers DocumentAsksObservers) DocumentAsksPerformed(
	ctx context.Context,
	performed Performed,
) {
	for _, observer := range observers {
		observer.DocumentAsksPerformed(ctx, performed)
	}
}
