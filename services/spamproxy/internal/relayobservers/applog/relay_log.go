// Package applog writes the facts of the relay to the application log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/pagerelay"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
)

type RelayLog struct{}

func (RelayLog) PageAssessed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
	assessmentDuration time.Duration,
) {
	slog.DebugContext(ctx, "page assessed",
		slog.String("address", address.String()),
		slog.String("verdict", assessment.Verdict()),
		slog.Float64("score", assessment.Score),
		slog.Duration("assessmentDuration", assessmentDuration),
	)
}

func (RelayLog) NonPageRelayed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason pagerelay.NonPageReason,
) {
	slog.DebugContext(ctx, "non-page relayed",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}

func (RelayLog) AnswerRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason pagerelay.RefusalReason,
) {
	slog.WarnContext(ctx, "answer refused",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}

func (RelayLog) RelayFailed(ctx context.Context, address canonicalurl.CanonicalURL, cause error) {
	slog.WarnContext(ctx, "relay failed",
		slog.String("address", address.String()),
		slog.Any("error", cause),
	)
}
