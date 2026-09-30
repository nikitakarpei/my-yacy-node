// Package applog writes the failed upstream requests to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/roundtrippers/failurereporting"
)

type FailureLog struct{}

func (FailureLog) RoundTripFailed(
	ctx context.Context,
	address string,
	step failurereporting.Step,
	cause error,
) {
	slog.WarnContext(ctx, "upstream request failed",
		slog.String("address", address),
		slog.String("step", string(step)),
		slog.Any("error", cause),
	)
}
