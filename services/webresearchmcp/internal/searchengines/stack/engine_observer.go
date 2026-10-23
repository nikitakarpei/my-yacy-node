package stack

import "context"

type EngineObserver interface {
	EngineFailed(ctx context.Context, engineName string, query string, cause error)
}

type EngineObservers []EngineObserver

func (observers EngineObservers) EngineFailed(
	ctx context.Context,
	engineName string,
	query string,
	cause error,
) {
	for _, observer := range observers {
		observer.EngineFailed(ctx, engineName, query, cause)
	}
}
