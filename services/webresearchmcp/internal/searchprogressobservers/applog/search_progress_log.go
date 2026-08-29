// Package applog writes a log line for each search the service serves or fails to serve, so
// an operator can read what a caller asked the search engine for and what came back. It is
// the only place that decides how a fact reads and at which level it is written.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgSearchServed = "search served"
	msgSearchFailed = "search failed, the caller gets an error"
)

type SearchProgressLog struct{}

func (SearchProgressLog) SearchServed(
	ctx context.Context,
	query string,
	searchResultCount int,
) {
	slog.DebugContext(ctx, msgSearchServed,
		slog.String("query", query),
		slog.Int("searchResultCount", searchResultCount),
	)
}

func (SearchProgressLog) SearchFailed(ctx context.Context, query string, cause error) {
	slog.WarnContext(ctx, msgSearchFailed,
		slog.String("query", query),
		slog.Any("error", cause),
	)
}
