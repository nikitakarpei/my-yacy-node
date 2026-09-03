package applog

import (
	"context"
	"log/slog"
	"time"
)

const msgRenderEndedWhileWaitingForCapacity = "render ended while waiting for capacity"

type RenderCapacityLog struct{}

func (RenderCapacityLog) RenderWaitedForCapacity(context.Context, string, time.Duration) {}

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
