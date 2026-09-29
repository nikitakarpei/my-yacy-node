package pagerelay

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
	NonPageRelayed(ctx context.Context, address canonicalurl.CanonicalURL, reason NonPageReason)
	AnswerRefused(ctx context.Context, address canonicalurl.CanonicalURL, reason RefusalReason)
	RelayFailed(ctx context.Context, address canonicalurl.CanonicalURL, cause error)
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

func (observers Observers) NonPageRelayed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason NonPageReason,
) {
	for _, observer := range observers {
		observer.NonPageRelayed(ctx, address, reason)
	}
}

func (observers Observers) AnswerRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason RefusalReason,
) {
	for _, observer := range observers {
		observer.AnswerRefused(ctx, address, reason)
	}
}

func (observers Observers) RelayFailed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.RelayFailed(ctx, address, cause)
	}
}
