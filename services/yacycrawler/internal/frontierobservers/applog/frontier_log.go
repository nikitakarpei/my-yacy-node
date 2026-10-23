// Package applog reports to the service log the frontier operations the crawl
// state storage refused.
package applog

import (
	"context"
	"log/slog"
	"time"
)

const (
	msgFrontierLeaseFailed       = "no page visit could be leased from the frontier"
	msgFrontierCountFailed       = "the page visits on the frontier could not be counted"
	msgFrontierAcknowledgeFailed = "a settled page visit could not be removed from the frontier"
	msgFrontierReturnFailed      = "a page visit could not be returned to the frontier"
)

type FrontierLog struct{}

func (FrontierLog) FrontierLeaseFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgFrontierLeaseFailed, slog.Any("error", err))
}

func (FrontierLog) FrontierCountFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgFrontierCountFailed, slog.Any("error", err))
}

func (FrontierLog) FrontierAcknowledgeFailed(ctx context.Context, identity string, err error) {
	slog.WarnContext(ctx, msgFrontierAcknowledgeFailed,
		slog.String("pageVisit", identity),
		slog.Any("error", err),
	)
}

func (FrontierLog) FrontierReturnFailed(
	ctx context.Context,
	identity string,
	delay time.Duration,
	err error,
) {
	slog.WarnContext(ctx, msgFrontierReturnFailed,
		slog.String("pageVisit", identity),
		slog.Duration("delay", delay),
		slog.Any("error", err),
	)
}
