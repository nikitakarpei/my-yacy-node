// Package distributioncycle runs one distribution pass: it asks
// replicashortfall which peers are missing a replica of each due posting, offers
// it to them through the couriers, and writes down what happened.
package distributioncycle

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerroster"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingcourier"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postinghandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferwait"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/replicashortfall"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	skipTooFewReachablePeers = "too_few_reachable_peers"
	skipShortfallUnread      = "shortfall_unread"
)

// ReplicaRecipients learns from each offer answer which peers can receive a
// replica now, so the next cycle can route around a peer that keeps refusing.
type ReplicaRecipients interface {
	OfferAnswered(
		peer yacymodel.Hash,
		outcome postingcourier.Outcome,
		requestedPause time.Duration,
	)
}

type Cycle struct {
	vault             *vault.Vault
	shortfall         *replicashortfall.Shortfall
	delivery          *OfferDelivery
	handoff           *postinghandoff.Handoff
	replicas          *postingreplicas.Replicas
	waits             *postingofferwait.Wait
	recipients        ReplicaRecipients
	bounds            postingofferwait.Bounds
	schedule          *postingschedule.Schedule
	roster            peerroster.Roster
	observer          Observer
	now               func() time.Time
	postingsPerCycle  int
	cycleInterval     time.Duration
	minReachablePeers int
}

//nolint:revive // argument-limit: fifteen explicit, independently-meaningful collaborators
func New(
	v *vault.Vault,
	shortfall *replicashortfall.Shortfall,
	delivery *OfferDelivery,
	handoff *postinghandoff.Handoff,
	replicas *postingreplicas.Replicas,
	waits *postingofferwait.Wait,
	recipients ReplicaRecipients,
	bounds postingofferwait.Bounds,
	schedule *postingschedule.Schedule,
	roster peerroster.Roster,
	observer Observer,
	now func() time.Time,
	postingsPerCycle int,
	cycleInterval time.Duration,
	minReachablePeers int,
) *Cycle {
	return &Cycle{
		vault:             v,
		shortfall:         shortfall,
		delivery:          delivery,
		handoff:           handoff,
		replicas:          replicas,
		waits:             waits,
		recipients:        recipients,
		bounds:            bounds,
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
	c.observeScheduledPostings(ctx)
	c.observeLongestOfferLateness(ctx)

	reachablePeers := c.roster.ReachablePeers(ctx)
	if len(reachablePeers) < c.minReachablePeers {
		slog.DebugContext(
			ctx,
			"distribution cycle skipped: too few reachable peers",
			slog.Int("reachablePeers", len(reachablePeers)),
			slog.Int("minReachablePeers", c.minReachablePeers),
		)
		c.observer.ObserveCycleSkipped(skipTooFewReachablePeers)

		return
	}

	due, err := c.shortfall.Due(ctx, c.postingsPerCycle, reachablePeers)
	if err != nil {
		slog.ErrorContext(ctx, "replica shortfall not read", slog.Any("error", err))
		c.observer.ObserveCycleSkipped(skipShortfallUnread)

		return
	}

	c.observer.ObservePostingsGone(len(due.Gone))
	for _, identity := range due.Gone {
		slog.DebugContext(ctx, "due posting gone from index",
			slog.String("word", identity.Word.String()),
			slog.String("url", identity.URL.String()))
	}

	offers := batchOffers(due.Offers)
	accepted, backoff := c.deliverOffers(ctx, offers)
	recipients := recipientsByPosting(accepted)

	handedOff, err := c.handoff.PostingsHeldByCloserPeers(
		ctx,
		offeredPostings(due.Offers),
		recipients,
	)
	if err != nil {
		slog.ErrorContext(ctx, "postings held by closer peers not read", slog.Any("error", err))
		handedOff = nil
	}

	c.commitCycle(ctx, due, accepted, handedOff, backoff)
}

func offeredPostings(
	replicaOffers []replicashortfall.ReplicaOffer,
) []postingschedule.Identity {
	identities := make([]postingschedule.Identity, 0, len(replicaOffers))
	for _, replicaOffer := range replicaOffers {
		identities = append(identities, postingschedule.Identity{
			Word: replicaOffer.Posting.WordHash,
			URL:  replicaOffer.Posting.URLHash,
		})
	}

	return identities
}

func (c *Cycle) commitCycle(
	ctx context.Context,
	due replicashortfall.Due,
	accepted []offer,
	handedOff []postingschedule.Identity,
	backoff *postingBackoff,
) {
	var dropped int
	err := c.vault.Update(ctx, func(tx *vault.Txn) error {
		var err error
		if dropped, err = c.dropStaleReplicas(tx, due.Stale); err != nil {
			return err
		}
		if err = c.recordAcceptedReplicas(tx, accepted); err != nil {
			return err
		}
		if err = c.handoff.HandOffPostings(ctx, tx, handedOff); err != nil {
			return err
		}

		return c.reschedulePostings(
			tx, keptOffers(due.Offers, handedOff), backoff, recipientsByPosting(accepted),
		)
	})
	if err != nil {
		slog.ErrorContext(ctx, "distribution cycle not written", slog.Any("error", err))

		return
	}

	c.observer.ObserveStaleReplicasDropped(dropped)
	c.observer.ObservePostingsHandedOff(len(handedOff))
}

func (c *Cycle) observeScheduledPostings(ctx context.Context) {
	scheduled, err := c.schedule.ScheduledPostings(ctx)
	if err != nil {
		slog.WarnContext(ctx, "scheduled postings not read", slog.Any("error", err))

		return
	}

	c.observer.ObserveScheduledPostings(scheduled)
}

func (c *Cycle) observeLongestOfferLateness(ctx context.Context) {
	earliest, found, err := c.schedule.EarliestOfferDueAt(ctx)
	if err != nil {
		slog.WarnContext(ctx, "earliest offer due time not read", slog.Any("error", err))

		return
	}
	if !found {
		c.observer.ObserveLongestOfferLateness(0)

		return
	}

	c.observer.ObserveLongestOfferLateness(max(c.now().Sub(earliest), 0))
}

func (c *Cycle) dropStaleReplicas(
	tx *vault.Txn,
	staleReplicas []replicashortfall.StaleReplicas,
) (int, error) {
	var dropped int
	for _, stale := range staleReplicas {
		posting, err := c.replicas.RecordDropped(
			tx, stale.Posting.Word, stale.Posting.URL, stale.Peers,
		)
		if err != nil {
			return 0, err
		}
		dropped += posting
	}

	return dropped, nil
}

func batchOffers(replicaOffers []replicashortfall.ReplicaOffer) []offer {
	batch := newOfferBatch()
	for _, replicaOffer := range replicaOffers {
		for _, seed := range replicaOffer.Seeds {
			batch.Add(seed, replicaOffer.Posting)
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
		c.recipients.OfferAnswered(peerOffer.Peer.Hash, receipt.Outcome, receipt.Backoff)
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

func (c *Cycle) recordAcceptedReplicas(tx *vault.Txn, accepted []offer) error {
	for _, peerOffer := range accepted {
		if err := c.replicas.RecordAccepted(
			tx,
			peerOffer.Peer.Hash,
			peerOffer.Postings,
		); err != nil {
			return err
		}
	}

	return nil
}

func recipientsByPosting(accepted []offer) map[postingschedule.Identity][]yacymodel.Hash {
	recipients := make(map[postingschedule.Identity][]yacymodel.Hash)
	for _, peerOffer := range accepted {
		for _, posting := range peerOffer.Postings {
			identity := postingschedule.Identity{Word: posting.WordHash, URL: posting.URLHash}
			recipients[identity] = append(recipients[identity], peerOffer.Peer.Hash)
		}
	}

	return recipients
}

func keptOffers(
	replicaOffers []replicashortfall.ReplicaOffer,
	handedOff []postingschedule.Identity,
) []replicashortfall.ReplicaOffer {
	kept := make([]replicashortfall.ReplicaOffer, 0, len(replicaOffers))
	for _, replicaOffer := range replicaOffers {
		identity := postingschedule.Identity{
			Word: replicaOffer.Posting.WordHash,
			URL:  replicaOffer.Posting.URLHash,
		}
		if !slices.Contains(handedOff, identity) {
			kept = append(kept, replicaOffer)
		}
	}

	return kept
}

func (c *Cycle) reschedulePostings(
	tx *vault.Txn,
	replicaOffers []replicashortfall.ReplicaOffer,
	backoff *postingBackoff,
	acceptedRecipients map[postingschedule.Identity][]yacymodel.Hash,
) error {
	for _, replicaOffer := range replicaOffers {
		identity := postingschedule.Identity{
			Word: replicaOffer.Posting.WordHash,
			URL:  replicaOffer.Posting.URLHash,
		}
		recipients := acceptedRecipients[identity]

		wait, err := c.offerWait(
			tx, identity, len(recipients) >= replicaOffer.ReplicasNeeded, backoff,
		)
		if err != nil {
			return err
		}

		at := c.now().Add(wait)
		if err := c.schedule.Reschedule(tx, identity.Word, identity.URL, at); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cycle) offerWait(
	tx *vault.Txn,
	identity postingschedule.Identity,
	redundancyMet bool,
	backoff *postingBackoff,
) (time.Duration, error) {
	if redundancyMet {
		return c.bounds.Longest, c.waits.Forget(tx, identity.Word, identity.URL)
	}

	widened, err := c.waits.Widen(tx, identity.Word, identity.URL, c.bounds)
	if err != nil {
		return 0, err
	}

	return max(widened, backoff.Longest(identity)), nil
}
