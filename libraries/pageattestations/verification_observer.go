package pageattestations

import "context"

type VerificationObserver interface {
	AndroidTrustUnloaded(ctx context.Context, cause error)
	NoSignerTrusted(ctx context.Context, reason error)
	RowNotAttested(ctx context.Context, address string, reason error)
}
