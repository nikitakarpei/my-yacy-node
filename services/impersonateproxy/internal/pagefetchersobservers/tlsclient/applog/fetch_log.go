// Package applog writes the page fetches to the application log.
package applog

import (
	"context"
	"log/slog"
)

type FetchLog struct{}

func (FetchLog) PageFetched(ctx context.Context, address string, status int) {
	slog.DebugContext(ctx, "page fetched",
		slog.String("address", address),
		slog.Int("status", status),
	)
}

func (FetchLog) FetchFailed(ctx context.Context, address string, cause error) {
	slog.WarnContext(ctx, "page fetch failed",
		slog.String("address", address),
		slog.Any("error", cause),
	)
}

func (FetchLog) FetchCancelled(ctx context.Context, address string) {
	slog.DebugContext(ctx, "page fetch cancelled", slog.String("address", address))
}
