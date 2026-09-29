// Package applog writes the failed upstream requests to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/upstreamrequest"
)

type UpstreamRequestLog struct{}

func (UpstreamRequestLog) Failed(
	ctx context.Context,
	address string,
	step upstreamrequest.Step,
	cause error,
) {
	slog.WarnContext(ctx, "upstream request failed",
		slog.String("address", address),
		slog.String("step", string(step)),
		slog.Any("error", cause),
	)
}
