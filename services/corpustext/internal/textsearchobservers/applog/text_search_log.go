// Package applog writes a log line for each search the text corpus answers or fails, so an
// operator can read what was searched and how long the search index took.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
)

const (
	msgSearchAnswered = "text search answered"
	msgSearchFailed   = "text search failed, the search index did not answer"
)

type TextSearchLog struct{}

func (TextSearchLog) SearchAnswered(
	ctx context.Context,
	search textsearch.Search,
	elapsed time.Duration,
	amountOfMatches int,
) {
	slog.DebugContext(ctx, msgSearchAnswered,
		searchAttributesOf(search),
		slog.Duration("searchDuration", elapsed),
		slog.Int("matches", amountOfMatches),
	)
}

func (TextSearchLog) SearchFailed(
	ctx context.Context,
	search textsearch.Search,
	elapsed time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgSearchFailed,
		searchAttributesOf(search),
		slog.Duration("searchDuration", elapsed),
		slog.Any("error", cause),
	)
}

func searchAttributesOf(search textsearch.Search) slog.Attr {
	return slog.Group("search",
		slog.String("query", search.Query),
		slog.String("language", search.Language),
		slog.Int("offset", search.Offset),
		slog.Int("limit", search.Limit),
	)
}
