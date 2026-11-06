// Package applog writes a log line each time the node finds the RWI stream of the pagerwi
// service unavailable, and when it opens it, so an operator can see why no page is indexed.
package applog

import (
	"context"
	"log/slog"
	"time"
)

const (
	msgPageRWIStreamUnavailable = "rwi stream of pagerwi unavailable, nothing indexed until it opens"
	msgPageRWIStreamOpened      = "rwi stream of pagerwi opened"
)

type PageRWIOpeningLog struct{}

func (PageRWIOpeningLog) StreamUnavailable(
	ctx context.Context,
	retryPause time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgPageRWIStreamUnavailable,
		slog.Duration("retryPause", retryPause),
		slog.Any("error", cause),
	)
}

func (PageRWIOpeningLog) StreamOpened(ctx context.Context) {
	slog.DebugContext(ctx, msgPageRWIStreamOpened)
}
