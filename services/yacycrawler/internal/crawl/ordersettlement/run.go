package ordersettlement

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/contextcancellation"
)

const (
	msgOrderDropped     = "crawl order dropped"
	msgOrderRedelivered = "crawl order redelivered after traversal failure"
)

//nolint:revive // argument-limit: the settlement loop's collaborators are all distinct ports.
func Run(
	ctx context.Context,
	deliveries <-chan DeliveredOrder,
	traversal OrderTraversal,
	observer Progress,
	clock Clock,
	ownershipInterval time.Duration,
) error {
	s := &settlement{
		observer:          observer,
		traversal:         traversal,
		clock:             clock,
		ownershipInterval: ownershipInterval,
	}
	for {
		select {
		case <-ctx.Done():
			return contextcancellation.Err(ctx)
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}
			s.settleDelivery(ctx, delivery, s.crawl(ctx, delivery))
			if err := contextcancellation.Err(ctx); err != nil {
				return err
			}
		}
	}
}

type settlement struct {
	observer          Progress
	traversal         OrderTraversal
	clock             Clock
	ownershipInterval time.Duration
}

func (s *settlement) crawl(ctx context.Context, delivery DeliveredOrder) error {
	s.observer.OrderReceived()

	heartbeatCtx, cancel := context.WithCancel(ctx)
	var heartbeat sync.WaitGroup
	if s.ownershipInterval > 0 {
		lease := ownershipLease{
			extend:   delivery.ExtendOwnership,
			interval: s.ownershipInterval,
			clock:    s.clock,
		}
		heartbeat.Go(func() {
			lease.Renew(heartbeatCtx)
		})
	}

	err := s.traversal.Traverse(ctx, delivery)
	cancel()
	heartbeat.Wait()

	if err != nil {
		return fmt.Errorf("traverse order %s: %w", delivery.Order.OrderID, err)
	}
	return nil
}

func (s *settlement) settleDelivery(
	ctx context.Context,
	delivery DeliveredOrder,
	crawlErr error,
) {
	if crawlErr != nil {
		s.observer.OrderRedelivered()
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
	s.observer.OrderCompleted()
}
