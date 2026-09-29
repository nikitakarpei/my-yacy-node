package requestrelay

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
)

type Observer interface {
	PageAssessed(
		ctx context.Context,
		address canonicalurl.CanonicalURL,
		assessment spamassessment.Assessment,
		assessmentDuration time.Duration,
	)
	AssessmentSkipped(ctx context.Context, address canonicalurl.CanonicalURL, reason SkipReason)
	RequestRefused(ctx context.Context, address canonicalurl.CanonicalURL, reason RefusalReason)
	AnswerReadingFailed(ctx context.Context, address canonicalurl.CanonicalURL, cause error)
	ReplyCutShort(ctx context.Context, address canonicalurl.CanonicalURL, cause error)
}

type Observers []Observer

func (observers Observers) PageAssessed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
	assessmentDuration time.Duration,
) {
	for _, observer := range observers {
		observer.PageAssessed(ctx, address, assessment, assessmentDuration)
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

func (observers Observers) AnswerReadingFailed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.AnswerReadingFailed(ctx, address, cause)
	}
}

func (observers Observers) ReplyCutShort(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ReplyCutShort(ctx, address, cause)
	}
}
