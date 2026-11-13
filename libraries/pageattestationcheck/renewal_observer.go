package pageattestationcheck

import "context"

type RenewalObserver interface {
	AndroidTrustUnloaded(ctx context.Context, cause error)
}
