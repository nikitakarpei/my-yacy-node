// Package applog reports to the service log the search engine calls that failed and the search
// pages that could not be rendered.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgSearchEngineCallFailed = "search engine call failed"
	msgPageRenderFailed       = "search page could not be rendered"
)

type SearchPageLog struct{}

func (SearchPageLog) SearchEngineCallFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgSearchEngineCallFailed, slog.Any("error", err))
}

func (SearchPageLog) PageRenderFailed(ctx context.Context, err error) {
	slog.ErrorContext(ctx, msgPageRenderFailed, slog.Any("error", err))
}
