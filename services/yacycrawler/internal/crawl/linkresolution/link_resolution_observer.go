package linkresolution

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type LinkResolutionObserver interface {
	BaseURLUnresolved(
		ctx context.Context,
		enclosingBaseURL canonicalurl.CanonicalURL,
		statedBaseURL string,
		cause error,
	)
	LinksUnresolved(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		links int,
	)
}

type LinkResolutionObservers []LinkResolutionObserver

func (observers LinkResolutionObservers) BaseURLUnresolved(
	ctx context.Context,
	enclosingBaseURL canonicalurl.CanonicalURL,
	statedBaseURL string,
	cause error,
) {
	for _, observer := range observers {
		observer.BaseURLUnresolved(ctx, enclosingBaseURL, statedBaseURL, cause)
	}
}

func (observers LinkResolutionObservers) LinksUnresolved(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	links int,
) {
	for _, observer := range observers {
		observer.LinksUnresolved(ctx, pageURL, links)
	}
}
