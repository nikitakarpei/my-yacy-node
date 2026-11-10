package attestedstatements

import "context"

type SigningObserver interface {
	KeyUnavailable(ctx context.Context, cause error)
	SignatureRefused(ctx context.Context, cause error)
	KeyEvidenceUnreadable(ctx context.Context, cause error)
	TimeStampUnavailable(ctx context.Context, cause error)
}
