package ordersettlement

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/contextcancellation"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawlcapability"
)

const (
	msgOrderDropped     = "crawl order dropped"
	msgOrderRedelivered = "crawl order redelivered after traversal failure"
)

type OrderRunner struct {
	observer          crawlcapability.RunProgress
	traversal         OrderTraversal
	clock             crawlcapability.Clock
	ownershipInterval time.Duration
}

func NewOrderRunner(
	observer crawlcapability.RunProgress,
	traversal OrderTraversal,
	clock crawlcapability.Clock,
	ownershipInterval time.Duration,
) *OrderRunner {
	return &OrderRunner{
		observer:          observer,
		traversal:         traversal,
		clock:             clock,
		ownershipInterval: ownershipInterval,
	}
}

func (r *OrderRunner) Run(
	ctx context.Context,
	deliveries <-chan crawlcapability.DeliveredOrder,
) error {
	for {
		select {
		case <-ctx.Done():
			return contextcancellation.Err(ctx)
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}
			r.settleDelivery(ctx, delivery, r.crawl(ctx, delivery))
			if err := contextcancellation.Err(ctx); err != nil {
				return err
			}
		}
	}
}

func (r *OrderRunner) crawl(ctx context.Context, delivery crawlcapability.DeliveredOrder) error {
	r.observer.OrderReceived()

	heartbeatCtx, cancel := context.WithCancel(ctx)
	var heartbeat sync.WaitGroup
	if r.ownershipInterval > 0 {
		lease := ownershipLease{
			extend:   delivery.ExtendOwnership,
			interval: r.ownershipInterval,
			clock:    r.clock,
		}
		heartbeat.Add(1)
		go func() {
			defer heartbeat.Done()
			lease.Renew(heartbeatCtx)
		}()
	}

	err := r.traversal.Traverse(ctx, delivery)
	cancel()
	heartbeat.Wait()

	if err != nil {
		return fmt.Errorf("traverse order %s: %w", delivery.Order.OrderID, err)
	}
	return nil
}

func (r *OrderRunner) settleDelivery(
	ctx context.Context,
	delivery crawlcapability.DeliveredOrder,
	crawlErr error,
) {
	if crawlErr != nil {
		r.observer.OrderRedelivered()
		slog.WarnContext(ctx, msgOrderRedelivered,
			slog.String("order", delivery.Order.OrderID),
			slog.Any("error", crawlErr),
		)
		if err := delivery.Retry(ctx); err != nil {
			slog.WarnContext(ctx, msgOrderDropped,
				slog.String("order", delivery.Order.OrderID),
				slog.Any("error", err),
			)
		}
		return
	}
	if err := delivery.Ack(ctx); err != nil {
		slog.WarnContext(ctx, msgOrderDropped,
			slog.String("order", delivery.Order.OrderID),
			slog.Any("error", err),
		)
		return
	}
	r.observer.OrderCompleted()
}
