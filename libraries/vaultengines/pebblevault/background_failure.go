package pebblevault

import (
	"context"
	"log/slog"
)

func reportBackgroundFailure(err error) {
	slog.ErrorContext(
		context.Background(),
		"storage background work failed",
		slog.Any("error", err),
	)
}
