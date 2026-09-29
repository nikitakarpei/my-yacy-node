package requestrelay

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
	RequestRefused(ctx context.Context, address canonicalurl.CanonicalURL, reason RefusalReason)
	UpstreamResponseFailed(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		failure UpstreamResponseFailure,
		cause error,
	)
	ResponseLeftIncomplete(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		incompleteResponseCause IncompleteResponseCause,
		cause error,
	)
	ClientClosedRequest(ctx context.Context, address canonicalurl.CanonicalURL)
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

func (observers Observers) RequestRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason RefusalReason,
) {
	for _, observer := range observers {
		observer.RequestRefused(ctx, address, reason)
	}
}

func (observers Observers) UpstreamResponseFailed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	failure UpstreamResponseFailure,
	cause error,
) {
	for _, observer := range observers {
		observer.UpstreamResponseFailed(ctx, address, failure, cause)
	}
}

func (observers Observers) ResponseLeftIncomplete(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	incompleteResponseCause IncompleteResponseCause,
	cause error,
) {
	for _, observer := range observers {
		observer.ResponseLeftIncomplete(ctx, address, incompleteResponseCause, cause)
	}
}

func (observers Observers) ClientClosedRequest(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.ClientClosedRequest(ctx, address)
	}
}
