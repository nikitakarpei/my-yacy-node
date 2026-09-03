package pagevisit

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageReadingObserver interface {
	IndexingRefusalEnforced(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	LinkDiscoveryRefusalEnforced(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	PageHTMLUnreadable(ctx context.Context, pageURL canonicalurl.CanonicalURL, cause error)
}

type PageReadingObservers []PageReadingObserver

func (observers PageReadingObservers) IndexingRefusalEnforced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.IndexingRefusalEnforced(ctx, pageURL)
	}
}

func (observers PageReadingObservers) LinkDiscoveryRefusalEnforced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.LinkDiscoveryRefusalEnforced(ctx, pageURL)
	}
}

func (observers PageReadingObservers) PageHTMLUnreadable(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.PageHTMLUnreadable(ctx, pageURL, cause)
	}
}
