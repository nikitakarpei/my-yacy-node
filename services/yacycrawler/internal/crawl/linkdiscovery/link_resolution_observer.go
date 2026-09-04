package linkdiscovery

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type LinkResolutionObserver interface {
	BaseURLUnresolved(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		statedBaseURL string,
		cause error,
	)
	LinksUnresolved(
		ctx context.Context,
		baseURL canonicalurl.CanonicalURL,
		links int,
	)
}

type LinkResolutionObservers []LinkResolutionObserver

func (observers LinkResolutionObservers) BaseURLUnresolved(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	statedBaseURL string,
	cause error,
) {
	for _, observer := range observers {
		observer.BaseURLUnresolved(ctx, pageURL, statedBaseURL, cause)
	}
}

func (observers LinkResolutionObservers) LinksUnresolved(
	ctx context.Context,
	baseURL canonicalurl.CanonicalURL,
	links int,
) {
	for _, observer := range observers {
		observer.LinksUnresolved(ctx, baseURL, links)
	}
}
