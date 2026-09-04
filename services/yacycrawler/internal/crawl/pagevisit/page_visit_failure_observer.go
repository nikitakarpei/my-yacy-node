package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageVisitFailureObserver interface {
	LastPageVisitUnreadable(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	PageHTMLUnreadable(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
}

type PageVisitFailureObservers []PageVisitFailureObserver

func (observers PageVisitFailureObservers) LastPageVisitUnreadable(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.LastPageVisitUnreadable(ctx, pageURL, cause)
	}
}

func (observers PageVisitFailureObservers) PageHTMLUnreadable(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.PageHTMLUnreadable(ctx, pageURL, cause)
	}
}
