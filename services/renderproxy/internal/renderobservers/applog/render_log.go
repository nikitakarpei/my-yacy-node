package applog

import (
	"context"
	"log/slog"
	"time"
)

const (
	msgRenderSucceeded    = "render succeeded"
	msgRenderTimedOut     = "render timed out"
	msgRenderCallerGaveUp = "render abandoned by the caller"
	msgRenderPageTooLarge = "rendered page too large"
	msgRenderFailed       = "render failed with an unexpected error"
)

type RenderLog struct{}

func (RenderLog) RenderSucceeded(ctx context.Context, targetURL string, _ time.Duration) {
	slog.DebugContext(ctx, msgRenderSucceeded, slog.String("url", targetURL))
}

func (RenderLog) RenderTimedOut(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgRenderTimedOut,
		slog.String("url", targetURL),
		slog.Duration("renderDuration", renderDuration),
		slog.Any("error", cause),
	)
}

func (RenderLog) RenderCallerGaveUp(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgRenderCallerGaveUp,
		slog.String("url", targetURL),
		slog.Duration("renderDuration", renderDuration),
		slog.Any("error", cause),
	)
}

func (RenderLog) RenderPageTooLarge(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgRenderPageTooLarge,
		slog.String("url", targetURL),
		slog.Duration("renderDuration", renderDuration),
		slog.Any("error", cause),
	)
}

func (RenderLog) RenderFailed(
	ctx context.Context,
	targetURL string,
	renderDuration time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgRenderFailed,
		slog.String("url", targetURL),
		slog.Duration("renderDuration", renderDuration),
		slog.Any("error", cause),
	)
}
