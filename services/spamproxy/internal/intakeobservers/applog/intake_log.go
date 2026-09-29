// Package applog writes the facts of the proxy intake to the application log.
package applog

import (
	"context"
	"log/slog"
)

type IntakeLog struct{}

func (IntakeLog) MethodRefused(ctx context.Context, method string) {
	slog.WarnContext(ctx, "request method refused", slog.String("method", method))
}

func (IntakeLog) TargetRefused(ctx context.Context, target string, cause error) {
	slog.WarnContext(ctx, "request target refused",
		slog.String("target", target),
		slog.Any("error", cause),
	)
}
