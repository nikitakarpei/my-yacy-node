package deadlineenforcing

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Observer interface {
	RequestRefused(ctx context.Context, address canonicalurl.CanonicalURL, reason RefusalReason)
}

type Observers []Observer

func (observers Observers) RequestRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason RefusalReason,
) {
	for _, observer := range observers {
		observer.RequestRefused(ctx, address, reason)
	}
}
