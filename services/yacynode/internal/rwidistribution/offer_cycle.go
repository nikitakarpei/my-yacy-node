package rwidistribution

import (
	"context"
	"log/slog"
	"time"
)

type postingCourier interface {
	Offer(ctx context.Context, offer postingOffer) offerOutcome
}

type offerCycle struct {
	builder          *batchBuilder
	courier          postingCourier
	schedule         *offerSchedule
	ledger           *replicaLedger
	now              func() time.Time
	postingsPerCycle int
	cycleInterval    time.Duration
	refreshInterval  time.Duration
	retryInterval    time.Duration
	redundancy       int
}

func (c *offerCycle) Run(ctx context.Context) {
	c.offerOnce(ctx)

	ticker := time.NewTicker(c.cycleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.offerOnce(ctx)
		}
	}
}

func (c *offerCycle) offerOnce(ctx context.Context) {
	plan, err := c.builder.Build(ctx, c.postingsPerCycle)
	if err != nil {
		slog.ErrorContext(ctx, "posting offer batch not built", slog.Any("error", err))

		return
	}

	c.reschedule(ctx, plan.Satisfied, c.refreshInterval)
	c.reschedule(ctx, plan.Stalled, c.retryInterval)

	touched := make(map[duePosting]time.Duration, len(plan.Offers))
	for _, offer := range plan.Offers {
		outcome := c.courier.Offer(ctx, offer)
		retryAfter := c.retryInterval
		if outcome.RetryAfter > 0 {
			retryAfter = outcome.RetryAfter
		}
		for _, posting := range offer.Postings {
			entry := duePosting{Word: posting.WordHash, URL: posting.URLHash.Hash()}
			if _, seen := touched[entry]; !seen {
				touched[entry] = retryAfter
			}
		}
	}

	c.rescheduleByOutcome(ctx, touched)
}

func (c *offerCycle) rescheduleByOutcome(
	ctx context.Context,
	touched map[duePosting]time.Duration,
) {
	now := c.now()
	for entry, retryAfter := range touched {
		replicas, err := c.ledger.Replicas(ctx, entry.Word, entry.URL)
		if err != nil {
			slog.WarnContext(ctx, "replicas not read for reschedule",
				slog.String("word", entry.Word.String()),
				slog.String("url", entry.URL.String()),
				slog.Any("error", err))

			continue
		}

		interval := retryAfter
		if len(replicas) >= c.redundancy {
			interval = c.refreshInterval
		}
		if err := c.schedule.Reschedule(ctx, entry.Word, entry.URL, now.Add(interval)); err != nil {
			slog.WarnContext(ctx, "posting not rescheduled",
				slog.String("word", entry.Word.String()),
				slog.String("url", entry.URL.String()),
				slog.Any("error", err))
		}
	}
}

func (c *offerCycle) reschedule(ctx context.Context, entries []duePosting, interval time.Duration) {
	at := c.now().Add(interval)
	for _, entry := range entries {
		if err := c.schedule.Reschedule(ctx, entry.Word, entry.URL, at); err != nil {
			slog.WarnContext(ctx, "posting not rescheduled",
				slog.String("word", entry.Word.String()),
				slog.String("url", entry.URL.String()),
				slog.Any("error", err))
		}
	}
}
