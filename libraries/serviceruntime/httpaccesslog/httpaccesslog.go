// Package httpaccesslog writes a log line for each request a service served.
// A request that failed is logged at WARN, a request that succeeded at DEBUG.
package httpaccesslog

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/httpobservation"
)

const (
	requestHandledMessage = "http request handled"
	requestFailedMessage  = "http request failed"
)

type AccessLog struct{}

func New() AccessLog {
	return AccessLog{}
}

func (AccessLog) ObserveRequest(ctx context.Context, served httpobservation.ServedRequest) {
	attrs := []any{
		"method", served.Method,
		"path", served.Path,
		"status", served.Status,
		"duration_ms", served.Duration.Milliseconds(),
	}
	if served.ResponseWriteError != nil {
		attrs = append(attrs, "error", served.ResponseWriteError)
		slog.WarnContext(ctx, requestFailedMessage, attrs...)

		return
	}
	if served.Status >= http.StatusBadRequest {
		slog.WarnContext(ctx, requestFailedMessage, attrs...)

		return
	}
	slog.DebugContext(ctx, requestHandledMessage, attrs...)
}
