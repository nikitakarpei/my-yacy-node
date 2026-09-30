// Package applog writes the facts of adding verdicts to pages to the
// application log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/verdictadding"
)

type VerdictLog struct{}

func (VerdictLog) PageAssessed(
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

func (VerdictLog) ReadingSlotWaited(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	slog.DebugContext(ctx, "reading slot waited",
		slog.String("address", address.String()),
		slog.Duration("slotWait", slotWait),
	)
}

func (VerdictLog) AssessmentSkipped(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason verdictadding.SkipReason,
) {
	slog.DebugContext(ctx, "assessment skipped",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}

func (VerdictLog) PageRefused(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	reason verdictadding.RefusalReason,
) {
	slog.WarnContext(ctx, "page refused",
		slog.String("address", address.String()),
		slog.String("reason", string(reason)),
	)
}

func (VerdictLog) BodyPrefixReadFailed(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, "body prefix read failed",
		slog.String("address", address.String()),
		slog.Any("error", cause),
	)
}
