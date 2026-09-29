package assessmentgate

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type Observer interface {
	SlotWaited(ctx context.Context, address canonicalurl.CanonicalURL, slotWait time.Duration)
	AssessmentFinished(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		pageSize int,
		assessmentDuration time.Duration,
	)
	AssessmentPanicked(ctx context.Context, address canonicalurl.CanonicalURL, panicValue any)
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

func (observers Observers) AssessmentFinished(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	pageSize int,
	assessmentDuration time.Duration,
) {
	for _, observer := range observers {
		observer.AssessmentFinished(ctx, address, pageSize, assessmentDuration)
	}
}

func (observers Observers) AssessmentPanicked(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	panicValue any,
) {
	for _, observer := range observers {
		observer.AssessmentPanicked(ctx, address, panicValue)
	}
}
