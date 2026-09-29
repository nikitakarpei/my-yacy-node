// Package applog writes the facts of the assessment gate to the application
// log.
package applog

import (
	"context"
	"log/slog"

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
