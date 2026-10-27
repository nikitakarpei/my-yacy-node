// Package applog reports to the service log the visits that could not reach visitcrawl.
package applog

import (
	"context"
	"log/slog"
)

const msgRelayFailed = "visit did not reach visitcrawl"

type VisitRelayLog struct{}

func (VisitRelayLog) RelayFailed(ctx context.Context, err error) {
	slog.WarnContext(ctx, msgRelayFailed, slog.Any("error", err))
}
