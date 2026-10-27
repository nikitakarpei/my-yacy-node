// Package applog reports to the service log the searches of a result source that failed.
package applog

import (
	"context"
	"log/slog"
)

const (
	msgCorpusTextSearchFailed = "corpustext search failed"
	msgYaCySearchFailed       = "YaCy network search failed"
)

type CorpusTextLog struct{}

func (CorpusTextLog) SearchFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgCorpusTextSearchFailed, slog.Any("error", err))
}

type YaCySearchLog struct{}

func (YaCySearchLog) SearchFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgYaCySearchFailed, slog.Any("error", err))
}
