// Package applog writes a log line each time the bridge finds the crawled pages of the
// crawler unavailable, and when it opens them, so an operator can see why no page is relayed.
package applog

import (
	"context"
	"log/slog"
	"time"
)

const (
	msgCrawledPagesUnavailable = "crawled pages unavailable, nothing relayed until they open"
	msgCrawledPagesOpened      = "crawled pages opened"
)

type CrawledPageOpeningLog struct{}

func (CrawledPageOpeningLog) StreamUnavailable(
	ctx context.Context,
	retryPause time.Duration,
	cause error,
) {
	slog.ErrorContext(ctx, msgCrawledPagesUnavailable,
		slog.Duration("retryPause", retryPause),
		slog.Any("error", cause),
	)
}

func (CrawledPageOpeningLog) StreamOpened(ctx context.Context) {
	slog.DebugContext(ctx, msgCrawledPagesOpened)
}
