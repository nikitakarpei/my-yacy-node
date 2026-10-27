package classificationgate

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Observer interface {
	SlotWaited(ctx context.Context, address canonicalurl.CanonicalURL, slotWait time.Duration)
	ClassificationFinished(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		pageSize int,
		classificationDuration time.Duration,
	)
	ClassificationPanicked(ctx context.Context, address canonicalurl.CanonicalURL, panicValue any)
}

type Observers []Observer

func (observers Observers) SlotWaited(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	for _, observer := range observers {
		observer.SlotWaited(ctx, address, slotWait)
	}
}

func (observers Observers) ClassificationFinished(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	pageSize int,
	classificationDuration time.Duration,
) {
	for _, observer := range observers {
		observer.ClassificationFinished(ctx, address, pageSize, classificationDuration)
	}
}

func (observers Observers) ClassificationPanicked(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	panicValue any,
) {
	for _, observer := range observers {
		observer.ClassificationPanicked(ctx, address, panicValue)
	}
}
