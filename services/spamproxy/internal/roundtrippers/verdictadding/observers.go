package verdictadding

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
)

type Observer interface {
	PageAssessed(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		assessment spamassessment.Assessment,
	)
	ReadingSlotWaited(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		slotWait time.Duration,
	)
	AssessmentSkipped(ctx context.Context, address canonicalurl.CanonicalURL, reason SkipReason)
	PageRefused(ctx context.Context, address canonicalurl.CanonicalURL, reason RefusalReason)
	BodyPrefixReadFailed(ctx context.Context, address canonicalurl.CanonicalURL, cause error)
}

type Observers []Observer

func (observers Observers) PageAssessed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
) {
	for _, observer := range observers {
		observer.PageAssessed(ctx, address, assessment)
	}
}

func (observers Observers) ReadingSlotWaited(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	for _, observer := range observers {
		observer.ReadingSlotWaited(ctx, address, slotWait)
	}
}

func (observers Observers) AssessmentSkipped(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason SkipReason,
) {
	for _, observer := range observers {
		observer.AssessmentSkipped(ctx, address, reason)
	}
}

func (observers Observers) PageRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason RefusalReason,
) {
	for _, observer := range observers {
		observer.PageRefused(ctx, address, reason)
	}
}

func (observers Observers) BodyPrefixReadFailed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.BodyPrefixReadFailed(ctx, address, cause)
	}
}
