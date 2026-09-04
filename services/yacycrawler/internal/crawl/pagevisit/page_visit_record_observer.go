package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageVisitRecordObserver interface {
	PageVisitNotRecorded(ctx context.Context, pageURL canonicalurl.CanonicalURL, cause error)
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
