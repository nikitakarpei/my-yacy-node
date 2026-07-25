package ordersettlement

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
)

const msgOwnershipLapsed = "crawl order ownership renewal failed"

type ownershipLease struct {
	delivery OrderDelivery
	interval time.Duration
	clock    clock.Clock
}

func (l ownershipLease) KeepRenewing(ctx context.Context) {
	for {
		if err := l.clock.Sleep(ctx, l.interval); err != nil {
			return
		}
		if err := l.delivery.ExtendOwnership(ctx); err != nil {
			slog.WarnContext(ctx, msgOwnershipLapsed, slog.Any("error", err))
		}
	}
}
