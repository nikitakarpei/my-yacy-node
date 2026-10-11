package jetstream

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageVisitRecordObserver interface {
	PageVisitNotRecorded(ctx context.Context, pageURL canonicalurl.CanonicalURL, cause error)
	LastPageVisitNotRead(ctx context.Context, pageURL canonicalurl.CanonicalURL, cause error)
}

type PageVisitRecordObservers []PageVisitRecordObserver

func (observers PageVisitRecordObservers) PageVisitNotRecorded(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.PageVisitNotRecorded(ctx, pageURL, cause)
	}
}

func (observers PageVisitRecordObservers) LastPageVisitNotRead(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.LastPageVisitNotRead(ctx, pageURL, cause)
	}
}
