// Package applog reports to the service log the crawler calls that failed and the console pages
// that could not be rendered.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgCrawlerCallFailed = "crawler API call failed"
	msgPageRenderFailed  = "console page could not be rendered"
)

type ConsoleLog struct{}

func (ConsoleLog) CrawlerCallFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgCrawlerCallFailed, slog.Any("error", err))
}

func (ConsoleLog) PageRenderFailed(ctx context.Context, screen string, err error) {
	slog.ErrorContext(ctx, msgPageRenderFailed,
		slog.String("screen", screen),
		slog.Any("error", err),
	)
}
