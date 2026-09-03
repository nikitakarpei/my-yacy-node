package applog

import (
	"context"
	"log/slog"
)

const msgProxyResponseDeliveryFailed = "write proxy response to client failed"

type ProxyResponseDeliveryLog struct{}

func (ProxyResponseDeliveryLog) ProxyResponseDelivered(context.Context, string) {}

func (ProxyResponseDeliveryLog) ProxyResponseDeliveryFailed(
	ctx context.Context,
	targetURL string,
	cause error,
) {
	slog.WarnContext(ctx, msgProxyResponseDeliveryFailed,
		slog.String("url", targetURL),
		slog.Any("error", cause),
	)
}
