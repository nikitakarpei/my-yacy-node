// Package applog reports to the service log the operator API requests the crawler could
// not answer.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgCrawlerAPIRequestFailed = "operator API request failed"
	msgCrawlerAPIAnswerFailed  = "operator API answer could not be written"
)

type CrawlerAPILog struct{}

func (CrawlerAPILog) CrawlerAPIRequestFailed(ctx context.Context, path string, err error) {
	slog.WarnContext(ctx, msgCrawlerAPIRequestFailed,
		slog.String("path", path),
		slog.Any("error", err),
	)
}

func (CrawlerAPILog) CrawlerAPIAnswerFailed(ctx context.Context, path string, err error) {
	slog.WarnContext(ctx, msgCrawlerAPIAnswerFailed,
		slog.String("path", path),
		slog.Any("error", err),
	)
}
