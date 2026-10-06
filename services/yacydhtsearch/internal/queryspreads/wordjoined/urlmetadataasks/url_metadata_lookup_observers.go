package urlmetadataasks

import "context"

type URLMetadataLookupObserver interface {
	URLMetadataLookupPerformed(ctx context.Context, performed Performed)
}

type URLMetadataLookupObservers []URLMetadataLookupObserver

func (observers URLMetadataLookupObservers) URLMetadataLookupPerformed(
	ctx context.Context,
	performed Performed,
) {
	for _, observer := range observers {
		observer.URLMetadataLookupPerformed(ctx, performed)
	}
}
