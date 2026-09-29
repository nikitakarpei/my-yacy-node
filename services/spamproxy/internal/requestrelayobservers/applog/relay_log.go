// Package applog writes the facts of the relay to the application log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
)

type RelayLog struct{}

func (RelayLog) PageAssessed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	assessment spamassessment.Assessment,
) {
	slog.DebugContext(ctx, "page assessed",
		slog.String("address", address.String()),
		slog.String("verdict", assessment.Verdict().String()),
		slog.Float64("score", assessment.Score),
	)
}

func (RelayLog) ReadingSlotWaited(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	slog.DebugContext(ctx, "reading slot waited",
		slog.String("address", address.String()),
		slog.Duration("slotWait", slotWait),
	)
}

func (RelayLog) AssessmentSkipped(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason requestrelay.SkipReason,
) {
	slog.DebugContext(ctx, "assessment skipped",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}

func (RelayLog) RequestRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason requestrelay.RefusalReason,
) {
	slog.WarnContext(ctx, "request refused",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}

func (RelayLog) AnswerReadingFailed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, "answer reading failed",
		slog.String("address", address.String()),
		slog.Any("error", cause),
	)
}

func (RelayLog) ReplyCutShort(ctx context.Context, address canonicalurl.CanonicalURL, cause error) {
	slog.WarnContext(ctx, "reply cut short",
		slog.String("address", address.String()),
		slog.Any("error", cause),
	)
}

func (RelayLog) ClientLeft(ctx context.Context, address canonicalurl.CanonicalURL) {
	slog.DebugContext(ctx, "client left", slog.String("address", address.String()))
}
