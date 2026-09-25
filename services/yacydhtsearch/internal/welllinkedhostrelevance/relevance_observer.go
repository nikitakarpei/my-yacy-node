package welllinkedhostrelevance

import "context"

type RelevanceObserver interface {
	DocumentsDemoted(ctx context.Context, amountOfDemotedDocuments int)
}

type RelevanceObservers []RelevanceObserver

func (observers RelevanceObservers) DocumentsDemoted(
	ctx context.Context,
	amountOfDemotedDocuments int,
) {
	for _, observer := range observers {
		observer.DocumentsDemoted(ctx, amountOfDemotedDocuments)
	}
}
