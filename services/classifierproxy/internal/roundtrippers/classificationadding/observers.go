package classificationadding

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

type Observer interface {
	PageClassified(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		classification pageclassification.Classification,
	)
	ReadingSlotWaited(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		slotWait time.Duration,
	)
	ClassificationSkipped(ctx context.Context, address canonicalurl.CanonicalURL, reason SkipReason)
	PageRefused(ctx context.Context, address canonicalurl.CanonicalURL, reason RefusalReason)
	BodyPrefixReadFailed(ctx context.Context, address canonicalurl.CanonicalURL, cause error)
}

type Observers []Observer

func (observers Observers) PageClassified(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	classification pageclassification.Classification,
) {
	for _, observer := range observers {
		observer.PageClassified(ctx, address, classification)
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

func (observers Observers) ClassificationSkipped(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason SkipReason,
) {
	for _, observer := range observers {
		observer.ClassificationSkipped(ctx, address, reason)
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
