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
		neverFailing(assembled.announcer.Run),
		neverFailing(func(ctx context.Context) {
			eviction.RunSweepLoop(
				ctx,
				assembled.sweeper,
				assembled.evictionObserver,
				evictionInterval,
			)
		}),
		neverFailing(func(ctx context.Context) {
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
		}),
	}
	if assembled.distributionCycle != nil {
		loops = append(loops, neverFailing(assembled.distributionCycle.Run))
	}
	if assembled.crawl != nil {
		loops = append(loops, neverFailing(assembled.crawl.Run))
	}

	return loops
}

func neverFailing(loop func(context.Context)) func(context.Context) error {
	return func(ctx context.Context) error {
		loop(ctx)

		return nil
	}
}
