package rwidistribution

import (
	"context"
	"log/slog"
	"time"
)

type postingCourier interface {
	Offer(ctx context.Context, offer postingOffer) postingOfferReceipt
}

type postingOfferCycle struct {
	planner          *postingOfferPlanner
	courier          postingCourier
	schedule         *postingOfferSchedule
	ledger           *replicaLedger
	observer         PostingOfferCycleObserver
	now              func() time.Time
	postingsPerCycle int
	cycleInterval    time.Duration
	refreshInterval  time.Duration
	retryInterval    time.Duration
	redundancy       int
}

func (c *postingOfferCycle) Run(ctx context.Context) {
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

func (c *postingOfferCycle) offerOnce(ctx context.Context) {
	plan, err := c.planner.Plan(ctx, c.postingsPerCycle)
	if err != nil {
		slog.ErrorContext(ctx, "posting offer plan not built", slog.Any("error", err))

		return
	}
	c.observer.ObserveScheduleDrain(plan.Drained)
	c.dropStaleReplicas(ctx, plan.StaleReplicas)

	offered := make(map[duePosting]time.Duration, len(plan.Offers))
	for _, offer := range plan.Offers {
		retryAfter := c.offer(ctx, offer)
		for _, posting := range offer.Postings {
			entry := duePosting{Word: posting.WordHash, URL: posting.URLHash}
			if _, seen := offered[entry]; !seen {
				offered[entry] = retryAfter
			}
		}
	}

	c.reschedule(ctx, c.dueTimes(ctx, plan, offered))
}

func (c *postingOfferCycle) offer(ctx context.Context, offer postingOffer) time.Duration {
	receipt := c.courier.Offer(ctx, offer)
	c.observer.ObservePostingOffer(string(receipt.Outcome), len(offer.Postings))

	if receipt.Outcome == postingOfferAccepted {
		c.recordAccepted(ctx, offer)
	}
	if receipt.RetryAfter > 0 {
		return receipt.RetryAfter
	}

	return c.retryInterval
}

func (c *postingOfferCycle) recordAccepted(ctx context.Context, offer postingOffer) {
	for _, posting := range offer.Postings {
		word, url := posting.WordHash, posting.URLHash
		if err := c.ledger.RecordAccepted(ctx, word, url, offer.Peer.Hash); err != nil {
			slog.WarnContext(ctx, "replica not recorded",
				slog.String("peer", offer.Peer.Hash.String()),
				slog.String("word", word.String()),
				slog.String("url", url.String()),
				slog.Any("error", err))
		}
	}
}

func (c *postingOfferCycle) dropStaleReplicas(ctx context.Context, stale []staleReplicas) {
	for _, entry := range stale {
		dropped, err := c.ledger.Drop(ctx, entry.Posting.Word, entry.Posting.URL, entry.Peers)
		if err != nil {
			slog.WarnContext(ctx, "stale replicas not dropped",
				slog.String("word", entry.Posting.Word.String()),
				slog.String("url", entry.Posting.URL.String()),
				slog.Any("error", err))

			continue
		}
		if dropped > 0 {
			c.observer.ObserveLedgerPrune(dropped)
		}
	}
}

func (c *postingOfferCycle) dueTimes(
	ctx context.Context,
	plan postingOfferPlan,
	offered map[duePosting]time.Duration,
) map[duePosting]time.Time {
	now := c.now()
	due := make(map[duePosting]time.Time, len(plan.Replicated)+len(plan.Unoffered)+len(offered))

	for _, posting := range plan.Replicated {
		due[posting] = now.Add(c.refreshInterval)
	}
	for _, posting := range plan.Unoffered {
		due[posting] = now.Add(c.retryInterval)
	}
	for posting, retryAfter := range offered {
		replicas, err := c.ledger.Replicas(ctx, posting.Word, posting.URL)
		if err != nil {
			slog.WarnContext(ctx, "replicas not read for reschedule",
				slog.String("word", posting.Word.String()),
				slog.String("url", posting.URL.String()),
				slog.Any("error", err))

			continue
		}

		interval := retryAfter
		if len(replicas) >= c.redundancy {
			interval = c.refreshInterval
		}
		due[posting] = now.Add(interval)
	}

	return due
}

func (c *postingOfferCycle) reschedule(ctx context.Context, due map[duePosting]time.Time) {
	for posting, at := range due {
		if err := c.schedule.Reschedule(ctx, posting.Word, posting.URL, at); err != nil {
			slog.WarnContext(ctx, "posting not rescheduled",
				slog.String("word", posting.Word.String()),
				slog.String("url", posting.URL.String()),
				slog.Any("error", err))
		}
	}
}
