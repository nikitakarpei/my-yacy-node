package rwidistribution

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
)

type postingOfferCycle struct {
	reader            *postingReplicationReader
	delivery          *postingOfferDelivery
	settlement        *postingOfferSettlement
	schedule          *postingOfferSchedule
	roster            peerroster.Roster
	observer          PostingOfferCycleObserver
	now               func() time.Time
	postingsPerCycle  int
	cycleInterval     time.Duration
	minReachablePeers int
}

func (c *postingOfferCycle) Run(ctx context.Context) {
	c.runCycle(ctx)

	ticker := time.NewTicker(c.cycleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.runCycle(ctx)
		}
	}
}

func (c *postingOfferCycle) runCycle(ctx context.Context) {
	c.observeOldestDuePostingAge(ctx)

	reachable := len(c.roster.ReachablePeers(ctx))
	if reachable < c.minReachablePeers {
		slog.DebugContext(
			ctx,
			"distribution cycle skipped: too few reachable peers",
			slog.Int("reachablePeers", reachable),
			slog.Int("minReachablePeers", c.minReachablePeers),
		)
		c.observer.ObserveCycleSkipped()

		return
	}

	due, err := c.reader.DueReplication(ctx, c.postingsPerCycle)
	if err != nil {
		slog.ErrorContext(ctx, "posting replication not read", slog.Any("error", err))

		return
	}
	c.observer.ObservePostingsDue(len(due.Postings))
	c.observer.ObservePostingsGone(len(due.Gone))
	for _, identity := range due.Gone {
		slog.DebugContext(ctx, "due posting gone from index",
			slog.String("word", identity.Word.String()),
			slog.String("url", identity.URL.String()))
	}

	byPeer := newPostingOffersByPeer()
	for _, replication := range due.Postings {
		for _, seed := range replication.SeedsMissingCopy {
			if !byPeer.Full(seed.Hash) {
				byPeer.Add(seed, replication.Posting)
			}
		}
	}

	retryAfters := newPostingRetryAfters()
	var accepted []postingOffer
	for _, offer := range byPeer.Offers() {
		result := c.delivery.Offer(ctx, offer)
		retryAfters.Record(offer, result.RetryAfter)
		if len(result.AcceptedPostings) > 0 {
			accepted = append(
				accepted,
				postingOffer{Peer: offer.Peer, Postings: result.AcceptedPostings},
			)
		}
	}

	c.settlement.Apply(ctx, due.Postings, accepted, retryAfters)
}

func (c *postingOfferCycle) observeOldestDuePostingAge(ctx context.Context) {
	oldest, found, err := c.schedule.OldestDueAt(ctx)
	if err != nil {
		slog.WarnContext(ctx, "oldest due posting not read", slog.Any("error", err))

		return
	}
	if !found {
		return
	}

	age := c.now().Sub(oldest)
	if age < 0 {
		age = 0
	}
	c.observer.ObserveOldestDuePostingAge(age)
}
