package urlmetadataaskceilings

import (
	"context"
	"time"
)

type AskCeilingObserver interface {
	AskCeilingSet(
		ctx context.Context,
		address string,
		pace time.Duration,
		ceiling int,
	)
}
