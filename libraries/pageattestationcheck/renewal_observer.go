package pageattestationcheck

import "context"

type RenewalObserver interface {
	TrustUnloaded(ctx context.Context, cause error)
}
