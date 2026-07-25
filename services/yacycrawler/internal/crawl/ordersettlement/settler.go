// Package ordersettlement owns each delivered crawl order until it is acknowledged or returned.
package ordersettlement

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/contextcancellation"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
)

const (
	msgOrderDropped     = "crawl order dropped"
	msgOrderRedelivered = "crawl order redelivered after traversal failure"
)

type Settler struct {
	traverser         OrderTraverser
	observer          OrderProgress
	clock             clock.Clock
	ownershipInterval time.Duration
}

func New(
	traverser OrderTraverser,
	observer OrderProgress,
	clock clock.Clock,
	ownershipInterval time.Duration,
) *Settler {
	return &Settler{
		traverser:         traverser,
		observer:          observer,
		clock:             clock,
		ownershipInterval: ownershipInterval,
	}
}

func (s *Settler) Settle(ctx context.Context, deliveries <-chan OrderDelivery) error {
	for {
		select {
		case <-ctx.Done():
			return contextcancellation.Err(ctx)
		case delivery, ok := <-deliveries:
			if !ok {
				return nil
			}
			s.settleDelivery(ctx, delivery, s.traverseUnderOwnership(ctx, delivery))
			if err := contextcancellation.Err(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *Settler) traverseUnderOwnership(ctx context.Context, delivery OrderDelivery) error {
	s.observer.OrderReceived()

	renewalCtx, cancel := context.WithCancel(ctx)
	var renewal sync.WaitGroup
	if s.ownershipInterval > 0 {
		lease := ownershipLease{
			delivery: delivery,
			interval: s.ownershipInterval,
			clock:    s.clock,
		}
		renewal.Go(func() {
			lease.KeepRenewing(renewalCtx)
		})
	}

	order := delivery.Order()
	err := s.traverser.Traverse(ctx, order)
	cancel()
	renewal.Wait()

	if err != nil {
		return fmt.Errorf("traverse order %s: %w", order.OrderID, err)
	}
	return nil
}

func (s *Settler) settleDelivery(
	ctx context.Context,
	delivery OrderDelivery,
	crawlErr error,
) {
	orderID := delivery.Order().OrderID
	if crawlErr != nil {
		s.observer.OrderRedelivered()
		slog.WarnContext(ctx, msgOrderRedelivered,
			slog.String("order", orderID),
			slog.Any("error", crawlErr),
		)
		if err := delivery.Return(ctx); err != nil {
			slog.WarnContext(ctx, msgOrderDropped,
				slog.String("order", orderID),
				slog.Any("error", err),
			)
		}
		return
	}
	if err := delivery.Acknowledge(ctx); err != nil {
		slog.WarnContext(ctx, msgOrderDropped,
			slog.String("order", orderID),
			slog.Any("error", err),
		)
		return
	}
	s.observer.OrderCompleted()
}
