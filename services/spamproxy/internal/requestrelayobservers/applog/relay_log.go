// Package applog writes the facts of the relay to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
)

type RelayLog struct{}

func (RelayLog) ResponseLeftIncomplete(
	ctx context.Context,
	address canonicalurl.CanonicalURL,
	incompleteResponseCause requestrelay.IncompleteResponseCause,
	cause error,
) {
	level := slog.LevelWarn
	if incompleteResponseCause == requestrelay.ClientClosedRequest {
		level = slog.LevelDebug
	}
	slog.Log(ctx, level, "response left incomplete",
		slog.String("address", address.String()),
		slog.String("cause", string(incompleteResponseCause)),
		slog.Any("error", cause),
	)
}

func (RelayLog) ClientClosedRequest(ctx context.Context, address canonicalurl.CanonicalURL) {
	slog.DebugContext(ctx, "client closed request", slog.String("address", address.String()))
}
