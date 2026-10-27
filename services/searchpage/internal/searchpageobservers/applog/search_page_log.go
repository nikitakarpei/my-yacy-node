// Package applog reports to the service log the search pages that could not be rendered.
package applog

import (
	"context"
	"log/slog"
)

const msgPageRenderFailed = "search page could not be rendered"

type SearchPageLog struct{}

func (SearchPageLog) PageRenderFailed(ctx context.Context, err error) {
	slog.ErrorContext(ctx, msgPageRenderFailed, slog.Any("error", err))
}
