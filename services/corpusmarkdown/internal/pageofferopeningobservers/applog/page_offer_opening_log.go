// Package applog writes a log line each time the markdown corpus finds the page offers of the
// scrape service unavailable, and when it opens them, so an operator can see why no page is
// stored.
package applog

import (
	"context"
	"log/slog"
	"time"
)

const (
	msgPageOffersUnavailable = "page offers unavailable, nothing stored until they open"
	msgPageOffersOpened      = "page offers opened"
)

type PageOfferOpeningLog struct{}

func (PageOfferOpeningLog) StreamUnavailable(
	ctx context.Context,
	retryPause time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgPageOffersUnavailable,
		slog.Duration("retryPause", retryPause),
		slog.Any("error", cause),
	)
}

func (PageOfferOpeningLog) StreamOpened(ctx context.Context) {
	slog.DebugContext(ctx, msgPageOffersOpened)
}
