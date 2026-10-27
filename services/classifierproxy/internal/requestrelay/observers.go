package requestrelay

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Observer interface {
	ResponseLeftIncomplete(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		incompleteResponseCause IncompleteResponseCause,
		cause error,
	)
	ClientClosedRequest(ctx context.Context, address canonicalurl.CanonicalURL)
}

type Observers []Observer

func (observers Observers) ResponseLeftIncomplete(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	incompleteResponseCause IncompleteResponseCause,
	cause error,
) {
	for _, observer := range observers {
		observer.ResponseLeftIncomplete(ctx, address, incompleteResponseCause, cause)
	}
}

func (observers Observers) ClientClosedRequest(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.ClientClosedRequest(ctx, address)
	}
}
