// Package applog writes the facts of the proxy intake to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/proxyintake"
)

type IntakeLog struct{}

func (IntakeLog) MethodRefused(ctx context.Context, method string) {
	slog.WarnContext(ctx, "request method refused", slog.String("method", method))
}

func (IntakeLog) TargetRefused(ctx context.Context, target string) {
	slog.WarnContext(ctx, "request target refused", slog.String("target", target))
}

func (IntakeLog) ResponseLeftIncomplete(
	ctx context.Context,
	address string,
	incompleteResponseCause proxyintake.IncompleteResponseCause,
	cause error,
) {
	level := slog.LevelWarn
	if incompleteResponseCause == proxyintake.ClientClosedRequest {
		level = slog.LevelDebug
	}
	slog.Log(ctx, level, "response left incomplete",
		slog.String("address", address),
		slog.String("cause", string(incompleteResponseCause)),
		slog.Any("error", cause),
	)
}
