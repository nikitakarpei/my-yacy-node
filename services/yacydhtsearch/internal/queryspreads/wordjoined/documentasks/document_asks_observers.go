package documentasks

import "context"

type DocumentAsksObserver interface {
	AskedAmongTheDocuments(
		ctx context.Context,
		decisionPerPartition DocumentsToMatchDecisionPerPartition,
	)
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
