package ordersettlement

import (
	"context"
	"log/slog"
	"time"
)

const msgOwnershipLapsed = "crawl order ownership heartbeat failed"

type ownershipLease struct {
	extend   func(context.Context) error
	interval time.Duration
	clock    Clock
}

func (l ownershipLease) Renew(ctx context.Context) {
	for {
		if err := l.clock.Sleep(ctx, l.interval); err != nil {
			return
		}
		if err := l.extend(ctx); err != nil {
			slog.WarnContext(ctx, msgOwnershipLapsed, slog.Any("error", err))
		}
	}
}
