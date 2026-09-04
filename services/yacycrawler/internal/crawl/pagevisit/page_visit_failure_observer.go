package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageVisitFailureObserver interface {
	PageHTMLUnreadable(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
}

type PageVisitFailureObservers []PageVisitFailureObserver

func (observers PageVisitFailureObservers) PageHTMLUnreadable(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.PageHTMLUnreadable(ctx, pageURL, cause)
	}
}
