// Package distributioncycle runs one distribution pass: it asks postingoffer
// which peers are missing a replica of each due posting, offers the posting
// to them, and writes down what happened.
package distributioncycle

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postinghandoff"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type SkipReason string

const SkipTooFewReachablePeers SkipReason = "too_few_reachable_peers"

type AbortReason string

const (
	AbortDuePostingsUnread AbortReason = "due_postings_unread"
	AbortBatchUnwritten    AbortReason = "not_written"
)

type ReachablePeers interface {
	ReachablePeers(ctx context.Context) []yacymodel.Seed
}

type Config struct {
	OfferInterval     postingofferschedule.OfferInterval
	PostingsPerBatch  int
	CycleInterval     time.Duration
	DrainBudget       time.Duration
	MinReachablePeers int
}

type Cycle struct {
	vault           *vault.Vault
	postingOffers   *postingoffer.PostingOffers
	handoff         *postinghandoff.Handoff
	transfers       *postingtransfer.PostingTransfers
	answers         OfferAnswers
	replicas        *postingreplicas.Replicas
	schedule        *postingofferschedule.Schedule
	roster          ReachablePeers
	now             func() time.Time
	cycleObserver   CycleObserver
	dhtRingObserver DHTRingObserver
	config          Config
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func New(
	v *vault.Vault,
	postingOffers *postingoffer.PostingOffers,
	handoff *postinghandoff.Handoff,
	transfers *postingtransfer.PostingTransfers,
	answers OfferAnswers,
	replicas *postingreplicas.Replicas,
	schedule *postingofferschedule.Schedule,
	roster ReachablePeers,
	now func() time.Time,
	cycleObserver CycleObserver,
	dhtRingObserver DHTRingObserver,
	config Config,
) *Cycle {
	return &Cycle{
		vault:           v,
		postingOffers:   postingOffers,
		handoff:         handoff,
		transfers:       transfers,
		answers:         answers,
		replicas:        replicas,
		schedule:        schedule,
		roster:          roster,
		now:             now,
		cycleObserver:   cycleObserver,
		dhtRingObserver: dhtRingObserver,
		config:          config,
	}
}

func (c *Cycle) Run(ctx context.Context) {
	c.runCycle(ctx)

	ticker := time.NewTicker(c.config.CycleInterval)
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
	c.schedule.ObserveBacklog(ctx)

	peers := c.roster.ReachablePeers(ctx)
	if len(peers) < c.config.MinReachablePeers {
		c.skipTooFewReachablePeers(ctx, len(peers))

		return
	}

	c.drainDuePostings(ctx, peers)
	c.cycleObserver.ObserveCycleCompleted()
}

func (c *Cycle) skipTooFewReachablePeers(ctx context.Context, reachablePeers int) {
	slog.DebugContext(
		ctx,
		"distribution cycle skipped: too few reachable peers",
		slog.Int("reachablePeers", reachablePeers),
		slog.Int("minReachablePeers", c.config.MinReachablePeers),
	)
	c.cycleObserver.ObserveCycleSkipped(string(SkipTooFewReachablePeers))
}

func (c *Cycle) drainDuePostings(ctx context.Context, peers []yacymodel.Seed) {
	drainEndsAt := c.now().Add(c.config.DrainBudget)

	for {
		duePostings, batchOffered := c.offerDueBatch(ctx, peers)
		if !batchOffered || duePostings < c.config.PostingsPerBatch {
			return
		}
		if ctx.Err() != nil || !c.now().Before(drainEndsAt) {
			return
		}
	}
}

func (c *Cycle) offerDueBatch(ctx context.Context, peers []yacymodel.Seed) (int, bool) {
	offers, gonePostings, err := c.postingOffers.DueNow(ctx, c.config.PostingsPerBatch, peers)
	if err != nil {
		c.reportDuePostingsUnread(ctx, err)

		return 0, false
	}
	c.reportPostingsGone(ctx, gonePostings)

	round := offerDuePostings(ctx, c.transfers, c.answers, offers)
	if err := c.commitBatch(ctx, offers, round); err != nil {
		c.reportBatchUnwritten(ctx, err)

		return 0, false
	}

	return len(offers) + len(gonePostings), true
}

func (c *Cycle) reportDuePostingsUnread(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "due postings not read", slog.Any("error", err))
	c.cycleObserver.ObserveBatchAborted(string(AbortDuePostingsUnread))
}

func (c *Cycle) reportPostingsGone(
	ctx context.Context,
	gonePostings []postingidentity.Identity,
) {
	c.cycleObserver.ObservePostingsGone(len(gonePostings))
	for _, identity := range gonePostings {
		slog.DebugContext(ctx, "due posting gone from index",
			slog.String("word", identity.Word.String()),
			slog.String("url", identity.URL.String()))
	}
}

func (c *Cycle) commitBatch(
	ctx context.Context,
	offers []postingoffer.PostingOffer,
	round offerRound,
) error {
	var (
		droppedReplicas   int
		handedOffPostings int
	)
	err := c.vault.Update(ctx, func(tx *vault.Txn) error {
		var err error
		if droppedReplicas, err = c.replicas.DropStaleHolders(
			tx,
			staleByPosting(offers),
		); err != nil {
			return err
		}
		if err = c.recordAcceptedReplicas(tx, round.acceptances); err != nil {
			return err
		}
		if handedOffPostings, err = c.handoff.HandOffPostingsHeldByCloserPeers(
			ctx, tx, offeredPostings(offers),
		); err != nil {
			return err
		}

		return c.setNextOffers(tx, offers, round)
	})
	if err != nil {
		return err
	}

	c.cycleObserver.ObserveStaleReplicasDropped(droppedReplicas)
	c.cycleObserver.ObservePostingsHandedOff(handedOffPostings)
	c.dhtRingObserver.ObserveReplicaRingFractions(
		replicaRingFractionsOf(offers, round.acceptances),
	)

	return nil
}

func replicaRingFractionsOf(
	offers []postingoffer.PostingOffer,
	acceptances []peerAcceptance,
) []float64 {
	offerByPosting := make(map[postingidentity.Identity]postingoffer.PostingOffer, len(offers))
	for _, offer := range offers {
		identity := postingidentity.IdentityOf(offer.Posting.WordHash, offer.Posting.URLHash)
		offerByPosting[identity] = offer
	}

	var ringFractions []float64
	for _, acceptance := range acceptances {
		for _, posting := range acceptance.postings {
			identity := postingidentity.IdentityOf(posting.WordHash, posting.URLHash)
			offer, offered := offerByPosting[identity]
			if !offered {
				continue
			}
			ringFractions = append(ringFractions, offer.PostingPosition.DistanceTo(
				yacymodel.DHTRingPositionOf(acceptance.holder),
			).FractionOfDHTRing())
		}
	}

	return ringFractions
}

func staleByPosting(
	offers []postingoffer.PostingOffer,
) map[postingidentity.Identity][]yacymodel.Hash {
	staleHolders := make(map[postingidentity.Identity][]yacymodel.Hash, len(offers))
	for _, offer := range offers {
		if len(offer.StaleHolders) == 0 {
			continue
		}
		identity := postingidentity.IdentityOf(offer.Posting.WordHash, offer.Posting.URLHash)
		staleHolders[identity] = append(staleHolders[identity], offer.StaleHolders...)
	}

	return staleHolders
}

func (c *Cycle) recordAcceptedReplicas(tx *vault.Txn, acceptances []peerAcceptance) error {
	for _, acceptance := range acceptances {
		if err := c.replicas.RecordAccepted(
			tx, acceptance.holder, acceptance.postings,
		); err != nil {
			return err
		}
	}

	return nil
}

func offeredPostings(offers []postingoffer.PostingOffer) []yacymodel.RWIPosting {
	postings := make([]yacymodel.RWIPosting, 0, len(offers))
	for _, offer := range offers {
		postings = append(postings, offer.Posting)
	}

	return postings
}

func (c *Cycle) setNextOffers(
	tx *vault.Txn,
	offers []postingoffer.PostingOffer,
	round offerRound,
) error {
	acceptances := acceptancesByPosting(round.acceptances)

	for _, offer := range offers {
		identity := postingidentity.IdentityOf(offer.Posting.WordHash, offer.Posting.URLHash)

		var err error
		if acceptances[identity] >= offer.AcceptancesNeeded {
			err = c.schedule.SetNextOfferAfterRedundancyMet(tx, identity, c.config.OfferInterval)
		} else {
			err = c.schedule.SetNextOfferAfterRedundancyMissed(
				tx, identity, c.config.OfferInterval, round.pauses[identity],
			)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func acceptancesByPosting(acceptances []peerAcceptance) map[postingidentity.Identity]int {
	byPosting := make(map[postingidentity.Identity]int)
	for _, acceptance := range acceptances {
		for _, posting := range acceptance.postings {
			byPosting[postingidentity.IdentityOf(posting.WordHash, posting.URLHash)]++
		}
	}

	return byPosting
}

func (c *Cycle) reportBatchUnwritten(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "distribution batch not written", slog.Any("error", err))
	c.cycleObserver.ObserveBatchAborted(string(AbortBatchUnwritten))
}
