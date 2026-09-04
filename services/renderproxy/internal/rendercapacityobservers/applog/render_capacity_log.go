package applog

import (
	"context"
	"log/slog"
	"time"
)

const (
	msgRenderWaitedForCapacity            = "render waited for capacity"
	msgRenderEndedWhileWaitingForCapacity = "render ended while waiting for capacity"
)

type RenderCapacityLog struct{}

func (RenderCapacityLog) RenderWaitedForCapacity(
	ctx context.Context,
	targetURL string,
	waitDuration time.Duration,
) {
	slog.DebugContext(ctx, msgRenderWaitedForCapacity,
		slog.String("url", targetURL),
		slog.Duration("waitDuration", waitDuration),
	)
}

func (RenderCapacityLog) RenderEndedWhileWaitingForCapacity(
	ctx context.Context,
	targetURL string,
	_ time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgRenderEndedWhileWaitingForCapacity,
		slog.String("url", targetURL),
		slog.Any("error", cause),
	)
}
