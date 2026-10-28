package pagereading

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageReadingObserver interface {
	MediaTypeUnparsed(ctx context.Context, contentType string, cause error)
	LinkDiscoveryRefusalEnforced(ctx context.Context, pageURL canonicalurl.CanonicalURL)
}

type PageReadingObservers []PageReadingObserver

func (observers PageReadingObservers) MediaTypeUnparsed(
	ctx context.Context,
	contentType string,
	cause error,
) {
	for _, observer := range observers {
		observer.MediaTypeUnparsed(ctx, contentType, cause)
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
