package rwidistribution

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
)

type postingCourier interface {
	Offer(ctx context.Context, offer postingOffer) postingOfferReceipt
}

type postingOfferCycle struct {
	reader            *postingReplicationReader
	courier           postingCourier
	schedule          *postingOfferSchedule
	ledger            *replicaLedger
	roster            peerroster.Roster
	observer          PostingOfferCycleObserver
	cadence           postingOfferCadence
	now               func() time.Time
	postingsPerCycle  int
	cycleInterval     time.Duration
	minReachablePeers int
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
	c.observer.ObservePostingsConsidered(len(due.Postings) + len(due.Gone))

	byPeer := newPostingOffersByPeer()
	for _, replication := range due.Postings {
		for _, seed := range replication.SeedsMissingCopy {
			if !byPeer.Full(seed.Hash) {
				byPeer.Add(seed, replication.Posting)
			}
		}
	}

	accepted := make(map[postingIdentity]int, len(due.Postings))
	retryAfter := make(map[postingIdentity]time.Duration, len(due.Postings))
	for _, offer := range byPeer.Offers() {
		receipt := c.courier.Offer(ctx, offer)
		c.observer.ObservePostingOffer(string(receipt.Outcome), len(offer.Postings))

		if receipt.Outcome == postingOfferAccepted {
			c.recordAccepted(ctx, offer)
		}
		for _, posting := range offer.Postings {
			id := postingIdentity{Word: posting.WordHash, URL: posting.URLHash}
			if receipt.Outcome == postingOfferAccepted {
				accepted[id]++
			}
			if receipt.RetryAfter > retryAfter[id] {
				retryAfter[id] = receipt.RetryAfter
			}
		}
	}

	c.dropStaleReplicas(ctx, due.Postings)
	c.forgetGone(ctx, due.Gone)
	c.reschedule(ctx, due.Postings, accepted, retryAfter)
}

func (c *postingOfferCycle) recordAccepted(ctx context.Context, offer postingOffer) {
	if err := c.ledger.RecordAccepted(ctx, offer); err != nil {
		slog.WarnContext(ctx, "replicas not recorded",
			slog.String("peer", offer.Peer.Hash.String()),
			slog.Any("error", err))
	}
}

func (c *postingOfferCycle) dropStaleReplicas(ctx context.Context, postings []postingReplication) {
	for _, replication := range postings {
		if len(replication.PeerHashesNoLongerResponsible) == 0 {
			continue
		}

		word, url := replication.Posting.WordHash, replication.Posting.URLHash
		dropped, err := c.ledger.RecordDropped(
			ctx, word, url, replication.PeerHashesNoLongerResponsible,
		)
		if err != nil {
			slog.WarnContext(ctx, "stale replicas not dropped",
				slog.String("word", word.String()),
				slog.String("url", url.String()),
				slog.Any("error", err))

			continue
		}
		if dropped > 0 {
			c.observer.ObserveLedgerPrune(dropped)
		}
	}
}

func (c *postingOfferCycle) forgetGone(ctx context.Context, gone []postingIdentity) {
	for _, identity := range gone {
		if err := c.schedule.Forget(ctx, identity.Word, identity.URL); err != nil {
			slog.WarnContext(ctx, "gone posting not forgotten",
				slog.String("word", identity.Word.String()),
				slog.String("url", identity.URL.String()),
				slog.Any("error", err))
		}
	}
}

func (c *postingOfferCycle) reschedule(
	ctx context.Context,
	postings []postingReplication,
	accepted map[postingIdentity]int,
	retryAfter map[postingIdentity]time.Duration,
) {
	now := c.now()
	for _, replication := range postings {
		id := postingIdentity{Word: replication.Posting.WordHash, URL: replication.Posting.URLHash}
		replicated := replication.CopiesNeeded == 0 || accepted[id] >= replication.CopiesNeeded
		at := c.cadence.NextDue(now, replicated, retryAfter[id])

		if err := c.schedule.Reschedule(ctx, id.Word, id.URL, at); err != nil {
			slog.WarnContext(ctx, "posting not rescheduled",
				slog.String("word", id.Word.String()),
				slog.String("url", id.URL.String()),
				slog.Any("error", err))
		}
	}
}
