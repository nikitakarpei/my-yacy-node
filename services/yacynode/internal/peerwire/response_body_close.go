package peerwire

import (
	"context"
	"io"
	"log/slog"
)

func closeResponseBody(ctx context.Context, body io.Closer, operation string) {
	if err := body.Close(); err != nil {
		slog.WarnContext(
			ctx,
			"response body close failed",
			slog.String("operation", operation),
			slog.Any("error", err),
		)
	}
}
