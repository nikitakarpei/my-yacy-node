package main

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/eviction"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiescrow"
)

const (
	evictionInterval = time.Minute

	escrowHoldFor        = 5 * time.Minute
	escrowExpiryBatch    = 256
	escrowExpiryInterval = time.Minute
)

func backgroundLoopsOf(assembled node) []func(context.Context) error {
	loops := []func(context.Context) error{
		func(ctx context.Context) error {
			assembled.announcer.Run(ctx)

			return nil
		},
		func(ctx context.Context) error {
			eviction.RunSweepLoop(
				ctx,
				assembled.sweeper,
				assembled.evictionObserver,
				evictionInterval,
			)

			return nil
		},
		func(ctx context.Context) error {
			rwiescrow.RunExpiryLoop(
				ctx,
				assembled.escrow,
				assembled.escrowObserver,
				rwiescrow.ExpiryConfig{
					HoldFor:  escrowHoldFor,
					Interval: escrowExpiryInterval,
					Batch:    escrowExpiryBatch,
				},
			)

			return nil
		},
	}
	if assembled.distributionCycle != nil {
		loops = append(loops, func(ctx context.Context) error {
			assembled.distributionCycle.Run(ctx)

			return nil
		})
	}
	if assembled.crawl != nil {
		loops = append(loops, func(ctx context.Context) error {
			assembled.crawl.Run(ctx)

			return nil
		})
	}

	return loops
}
