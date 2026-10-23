// Package applog writes a log line for each search engine of the stack that fails a search,
// so an operator can tell which engine left its results out of an answer.
package applog

import (
	"context"
	"log/slog"
)

const msgEngineFailed = "search engine failed, the answer leaves its results out"

type EngineLog struct{}

func (EngineLog) EngineFailed(
	ctx context.Context,
	engineName string,
	query string,
	cause error,
) {
	slog.WarnContext(ctx, msgEngineFailed,
		slog.String("engine", engineName),
		slog.String("query", query),
		slog.Any("error", cause),
	)
}
