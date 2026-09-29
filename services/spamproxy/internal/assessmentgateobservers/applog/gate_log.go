// Package applog writes the facts of the assessment gate to the application
// log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type GateLog struct{}

func (GateLog) AssessmentPanicked(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	panicValue any,
) {
	slog.ErrorContext(ctx, "assessment panicked",
		slog.String("address", address.String()),
		slog.Any("panic", panicValue),
	)
}

func (GateLog) SlotWaited(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	slog.DebugContext(ctx, "assessment slot waited",
		slog.String("address", address.String()),
		slog.Duration("slotWait", slotWait),
	)
}

func (GateLog) AssessmentFinished(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	pageSize int,
	assessmentDuration time.Duration,
) {
	slog.DebugContext(ctx, "assessment finished",
		slog.String("address", address.String()),
		slog.Int("pageSize", pageSize),
		slog.Duration("assessmentDuration", assessmentDuration),
	)
}
