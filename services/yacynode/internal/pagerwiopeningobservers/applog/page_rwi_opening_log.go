// Package applog writes a log line each time the node finds the RWIs of the pagerwi service
// unavailable, and when it opens them, so an operator can see why no page is indexed.
package applog

import (
	"context"
	"log/slog"
	"time"
)

const (
	msgPageRWIsUnavailable = "rwis of pages unavailable, nothing indexed until they open"
	msgPageRWIsOpened      = "rwis of pages opened"
)

type PageRWIOpeningLog struct{}

func (PageRWIOpeningLog) StreamUnavailable(
	ctx context.Context,
	retryPause time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgPageRWIsUnavailable,
		slog.Duration("retryPause", retryPause),
		slog.Any("error", cause),
	)
}

func (PageRWIOpeningLog) StreamOpened(ctx context.Context) {
	slog.DebugContext(ctx, msgPageRWIsOpened)
}
