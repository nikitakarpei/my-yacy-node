// Package applog writes a log line each time the text corpus finds the page offers of the
// scrape service unavailable, and when it opens them, so an operator can see why no page is
// indexed.
package applog

import (
	"context"
	"log/slog"
	"time"
)

const (
	msgPageOffersUnavailable = "page offers unavailable, nothing indexed until they open"
	msgPageOffersOpened      = "page offers opened"
)

type PageOfferWaitLog struct{}

func (PageOfferWaitLog) PageOffersUnavailable(
	ctx context.Context,
	retryPause time.Duration,
	cause error,
) {
	slog.ErrorContext(ctx, msgPageOffersUnavailable,
		slog.Duration("retryPause", retryPause),
		slog.Any("error", cause),
	)
}

func (PageOfferWaitLog) PageOffersOpened(ctx context.Context) {
	slog.DebugContext(ctx, msgPageOffersOpened)
}
