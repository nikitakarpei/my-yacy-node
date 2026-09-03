package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/rendergate"
)

const (
	msgRenderSucceeded = "render succeeded"
	msgRenderFailed    = "render failed"
)

type RenderLog struct{}

func (RenderLog) RenderSucceeded(ctx context.Context, targetURL string, _ time.Duration) {
	slog.DebugContext(ctx, msgRenderSucceeded, slog.String("url", targetURL))
}

func (RenderLog) RenderFailed(
	ctx context.Context,
	targetURL string,
	_ time.Duration,
	reason rendergate.RenderFailureReason,
	cause error,
) {
	slog.WarnContext(ctx, msgRenderFailed,
		slog.String("url", targetURL),
		slog.String("reason", string(reason)),
		slog.Any("error", cause),
	)
}
