// Package distributioncycle runs one distribution pass: it asks
// replicashortfall which peers are missing a replica of each due posting, offers
// it to them through the couriers, and writes down what happened.
package distributioncycle

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicashortfall"
)

type Cycle struct {
	shortfall         *replicashortfall.Shortfall
	delivery          *OfferDelivery
	replicas          *postingreplicas.Replicas
	cadence           Cadence
	schedule          *postingschedule.Schedule
	roster            peerroster.Roster
	observer          Observer
	now               func() time.Time
	postingsPerCycle  int
	cycleInterval     time.Duration
	minReachablePeers int
}

//nolint:revive // argument-limit: eleven explicit, independently-meaningful collaborators
func New(
	shortfall *replicashortfall.Shortfall,
	delivery *OfferDelivery,
	replicas *postingreplicas.Replicas,
	cadence Cadence,
	schedule *postingschedule.Schedule,
	roster peerroster.Roster,
	observer Observer,
	now func() time.Time,
	postingsPerCycle int,
	cycleInterval time.Duration,
	minReachablePeers int,
) *Cycle {
	return &Cycle{
		shortfall:         shortfall,
		delivery:          delivery,
		replicas:          replicas,
		cadence:           cadence,
		schedule:          schedule,
		roster:            roster,
		observer:          observer,
		now:               now,
		postingsPerCycle:  postingsPerCycle,
		cycleInterval:     cycleInterval,
		minReachablePeers: minReachablePeers,
	}
}

func (c *Cycle) Run(ctx context.Context) {
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

func (c *Cycle) runCycle(ctx context.Context) {
	c.observeOldestDuePostingAge(ctx)

	reachablePeers := c.roster.ReachablePeers(ctx)
	if len(reachablePeers) < c.minReachablePeers {
		slog.DebugContext(
			ctx,
			"distribution cycle skipped: too few reachable peers",
			slog.Int("reachablePeers", len(reachablePeers)),
			slog.Int("minReachablePeers", c.minReachablePeers),
		)
		c.observer.ObserveCycleSkipped()

		return
	}

	due, err := c.shortfall.Due(ctx, c.postingsPerCycle, reachablePeers)
	if err != nil {
		slog.ErrorContext(ctx, "replica shortfall not read", slog.Any("error", err))
		c.observer.ObserveShortfallUnread()

		return
	}

	c.observer.ObservePostingsGone(len(due.Gone))
	for _, identity := range due.Gone {
		slog.DebugContext(ctx, "due posting gone from index",
			slog.String("word", identity.Word.String()),
			slog.String("url", identity.URL.String()))
	}

	c.dropStaleReplicas(ctx, due.Stale)

	offers := batchOffers(due.Missing)
	accepted, backoff := c.deliverOffers(ctx, offers)
	recorded := c.recordAcceptedReplicas(ctx, accepted)
	c.reschedulePostings(ctx, due.Missing, backoff, replicasByPosting(recorded))
}

func (c *Cycle) observeOldestDuePostingAge(ctx context.Context) {
	oldest, found, err := c.schedule.OldestDueAt(ctx)
	if err != nil {
		slog.WarnContext(ctx, "oldest due posting not read", slog.Any("error", err))

		return
	}
	if !found {
		c.observer.ObserveOldestDuePostingAge(0)

		return
	}

	age := max(c.now().Sub(oldest), 0)
	c.observer.ObserveOldestDuePostingAge(age)
}

func (c *Cycle) dropStaleReplicas(
	ctx context.Context,
	staleReplicas []replicashortfall.StaleReplicas,
) {
	for _, stale := range staleReplicas {
		word, url := stale.Posting.Word, stale.Posting.URL
		dropped, err := c.replicas.RecordDropped(ctx, word, url, stale.Peers)
		if err != nil {
			slog.WarnContext(ctx, "stale replicas not dropped",
				slog.String("word", word.String()),
				slog.String("url", url.String()),
				slog.Any("error", err))

			continue
		}
		if dropped > 0 {
			c.observer.ObserveStaleReplicasDropped(dropped)
		}
	}
}

func batchOffers(missingReplicas []replicashortfall.MissingReplicas) []offer {
	batch := newOfferBatch()
	for _, missing := range missingReplicas {
		for _, seed := range missing.Seeds {
			batch.Add(seed, missing.Posting)
		}
	}

	return batch.Offers()
}

func (c *Cycle) deliverOffers(
	ctx context.Context,
	offers []offer,
) ([]offer, *postingBackoff) {
	backoff := newPostingBackoff()

	var accepted []offer
	for _, peerOffer := range offers {
		receipt := c.delivery.Offer(ctx, peerOffer)
		backoff.Record(peerOffer, receipt.Backoff)
		if len(receipt.AcceptedPostings) > 0 {
			accepted = append(
				accepted,
				offer{Peer: peerOffer.Peer, Postings: receipt.AcceptedPostings},
			)
		}
	}

	return accepted, backoff
}

func (c *Cycle) recordAcceptedReplicas(ctx context.Context, accepted []offer) []offer {
	var recorded []offer
	for _, peerOffer := range accepted {
		err := c.replicas.RecordAccepted(ctx, peerOffer.Peer.Hash, peerOffer.Postings)
		if err != nil {
			slog.WarnContext(ctx, "replicas not recorded",
				slog.String("peer", peerOffer.Peer.Hash.String()),
				slog.Any("error", err))

			continue
		}
		recorded = append(recorded, peerOffer)
	}

	return recorded
}

func replicasByPosting(recorded []offer) map[postingschedule.Identity]int {
	replicas := make(map[postingschedule.Identity]int)
	for _, peerOffer := range recorded {
		for _, posting := range peerOffer.Postings {
			identity := postingschedule.Identity{Word: posting.WordHash, URL: posting.URLHash}
			replicas[identity]++
		}
	}

	return replicas
}

func (c *Cycle) reschedulePostings(
	ctx context.Context,
	missingReplicas []replicashortfall.MissingReplicas,
	backoff *postingBackoff,
	recordedReplicas map[postingschedule.Identity]int,
) {
	for _, missing := range missingReplicas {
		identity := postingschedule.Identity{
			Word: missing.Posting.WordHash,
			URL:  missing.Posting.URLHash,
		}
		redundancyMet := recordedReplicas[identity] >= missing.ReplicasNeeded
		at := c.cadence.NextDue(c.now(), redundancyMet, backoff.Longest(identity))

		if err := c.schedule.Reschedule(ctx, identity.Word, identity.URL, at); err != nil {
			slog.WarnContext(ctx, "posting not rescheduled",
				slog.String("word", identity.Word.String()),
				slog.String("url", identity.URL.String()),
				slog.Any("error", err))
		}
	}
}
