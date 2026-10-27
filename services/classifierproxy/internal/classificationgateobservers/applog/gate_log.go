// Package applog writes the facts of the classification gate to the application
// log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type GateLog struct{}

func (GateLog) ClassificationPanicked(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	panicValue any,
) {
	slog.ErrorContext(ctx, "classification panicked",
		slog.String("address", address.String()),
		slog.Any("panic", panicValue),
	)
}

func (GateLog) SlotWaited(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	slotWait time.Duration,
) {
	slog.DebugContext(ctx, "classification slot waited",
		slog.String("address", address.String()),
		slog.Duration("slotWait", slotWait),
	)
}

func (GateLog) ClassificationFinished(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	pageSize int,
	classificationDuration time.Duration,
) {
	slog.DebugContext(ctx, "classification finished",
		slog.String("address", address.String()),
		slog.Int("pageSize", pageSize),
		slog.Duration("classificationDuration", classificationDuration),
	)
}
