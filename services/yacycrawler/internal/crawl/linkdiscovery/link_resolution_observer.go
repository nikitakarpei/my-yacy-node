package linkdiscovery

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type LinkResolutionObserver interface {
	BaseHrefUnresolved(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		baseHref string,
		cause error,
	)
	LinkHrefsUnresolved(
		ctx context.Context,
		baseURL canonicalurl.CanonicalURL,
		hrefs int,
	)
}

type LinkResolutionObservers []LinkResolutionObserver

func (observers LinkResolutionObservers) BaseHrefUnresolved(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	baseHref string,
	cause error,
) {
	for _, observer := range observers {
		observer.BaseHrefUnresolved(ctx, pageURL, baseHref, cause)
	}
}

func (observers LinkResolutionObservers) LinkHrefsUnresolved(
	ctx context.Context,
	baseURL canonicalurl.CanonicalURL,
	hrefs int,
) {
	for _, observer := range observers {
		observer.LinkHrefsUnresolved(ctx, baseURL, hrefs)
	}
}
