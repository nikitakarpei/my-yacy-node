package pageattestations

import "context"

type RenewalObserver interface {
	AndroidTrustUnloaded(ctx context.Context, cause error)
	RenewalRefused(ctx context.Context, reason error)
}
