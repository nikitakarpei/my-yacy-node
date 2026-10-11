// Package distributioncycle runs one distribution pass: it asks postingoffer
// which peers are missing a replica of each due posting, offers the posting
// to them, and writes down what happened. Batches take the ring sectors in
// turn, several at once, each in a sector no other batch in flight holds; a
// batch offers to all its peers at once.
package distributioncycle

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingidentity"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferschedule"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingreplicas"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
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

type PostingHandoff interface {
	HandOffIfHeldByCloserPeers(
		ctx context.Context,
		tx *vault.Txn,
		posting yacymodel.RWIPosting,
		position yacymodel.DHTRingPosition,
		holders []yacymodel.Hash,
	) (bool, error)
}

type PostingTransfers interface {
	Send(
		ctx context.Context,
		peer yacymodel.Seed,
		offeredPostings []yacymodel.RWIPosting,
	) postingtransfer.OfferAnswer
}

type Config struct {
	PostingsPerBatch  int
	BatchConcurrency  int
	CycleInterval     time.Duration
	DrainBudget       time.Duration
	MinReachablePeers int
}

type Cycle struct {
	vault           *vault.Vault
	postingOffers   *postingoffer.PostingOffers
	handoff         PostingHandoff
	transfers       PostingTransfers
	answers         OfferAnswers
	replicas        *postingreplicas.Replicas
	schedule        *postingofferschedule.Schedule
	roster          ReachablePeers
	now             func() time.Time
	cycleObserver   CycleObserver
	dhtRingObserver DHTRingObserver
	sectorTurns     *sectorTurns
	config          Config
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func New(
	v *vault.Vault,
	postingOffers *postingoffer.PostingOffers,
	handoff PostingHandoff,
	transfers PostingTransfers,
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
		sectorTurns:     &sectorTurns{},
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
	c.observeBacklog(ctx)

	peers := c.roster.ReachablePeers(ctx)
	if len(peers) < c.config.MinReachablePeers {
		c.skipTooFewReachablePeers(ctx, len(peers))

		return
	}

	c.drainDuePostings(ctx, peers)
	c.cycleObserver.ObserveCycleCompleted()
}

func (c *Cycle) observeBacklog(ctx context.Context) {
	if err := c.vault.View(ctx, func(tx *vault.Txn) error {
		return c.schedule.ObserveBacklog(tx)
	}); err != nil {
		slog.WarnContext(ctx, "offer backlog not observed", slog.Any("error", err))
	}
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
	c.sectorTurns.forgetDrainedSectors()

	var drains sync.WaitGroup
	for range c.config.BatchConcurrency {
		drains.Go(func() { c.offerBatchesUntil(ctx, peers, drainEndsAt) })
	}
	drains.Wait()
}

func (c *Cycle) offerBatchesUntil(
	ctx context.Context,
	peers []yacymodel.Seed,
	drainEndsAt time.Time,
) {
	for c.offerDueBatch(ctx, peers) {
		if ctx.Err() != nil || !c.now().Before(drainEndsAt) {
			return
		}
	}
}

func (c *Cycle) offerDueBatch(ctx context.Context, peers []yacymodel.Seed) bool {
	sector, found := c.sectorTurns.claimNext()
	if !found {
		return false
	}
	defer c.sectorTurns.release(sector)

	offers, err := c.postingOffers.DueNowIn(
		ctx, sector, c.config.PostingsPerBatch, peers,
	)
	if err != nil {
		c.reportDuePostingsUnread(ctx, err)

		return false
	}
	if len(offers) < c.config.PostingsPerBatch {
		c.sectorTurns.markDrained(sector)
	}
	if len(offers) == 0 {
		return true
	}

	round := offerDuePostings(ctx, c.transfers, c.answers, offers)
	if err := c.commitBatch(ctx, offers, round); err != nil {
		c.reportBatchUnwritten(ctx, err)

		return false
	}

	return true
}

func (c *Cycle) reportDuePostingsUnread(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "due postings not read", slog.Any("error", err))
	c.cycleObserver.ObserveBatchAborted(string(AbortDuePostingsUnread))
}

func (c *Cycle) commitBatch(
	ctx context.Context,
	offers []postingoffer.PostingOffer,
	round offerRound,
) error {
	err := c.vault.Update(ctx, func(tx *vault.Txn) error {
		for _, offer := range offers {
			answers := round[postingidentity.IdentityOf(offer.Posting)]
			if err := c.recordOfferOutcome(ctx, tx, offer, answers); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	c.dhtRingObserver.ObserveReplicaRingFractions(replicaRingFractionsOf(offers, round))

	return nil
}

func (c *Cycle) recordOfferOutcome(
	ctx context.Context,
	tx *vault.Txn,
	offer postingoffer.PostingOffer,
	answers postingAnswers,
) error {
	identity := postingidentity.IdentityOf(offer.Posting)
	recordedHolders, err := c.replicas.HoldersOf(tx, identity)
	if err != nil {
		return err
	}
	change := postingreplicas.HolderChange{
		StaleHolders:   offer.StaleHolders,
		AcceptingPeers: answers.acceptingPeers,
	}
	holders := change.HoldersFrom(recordedHolders)
	droppedReplicas := change.AmountOfStaleHoldersIn(recordedHolders)

	wasHandedOff, err := c.handoff.HandOffIfHeldByCloserPeers(
		ctx, tx, offer.Posting, offer.PostingPosition, holders,
	)
	if err != nil {
		return err
	}
	if wasHandedOff {
		c.observeStaleReplicasDropped(tx, droppedReplicas)
		c.observePostingHandedOff(tx)

		return nil
	}

	wasRescheduled, err := c.setNextOffer(tx, identity, offer.AcceptancesNeeded, answers)
	if err != nil || !wasRescheduled {
		return err
	}
	c.observeStaleReplicasDropped(tx, droppedReplicas)

	return c.replicas.RecordHolders(tx, identity, recordedHolders, holders)
}

func (c *Cycle) observeStaleReplicasDropped(tx *vault.Txn, droppedReplicas int) {
	tx.RunAfterCommit(func() { c.cycleObserver.ObserveStaleReplicasDropped(droppedReplicas) })
}

func (c *Cycle) observePostingHandedOff(tx *vault.Txn) {
	tx.RunAfterCommit(func() { c.cycleObserver.ObservePostingHandedOff() })
}

func (c *Cycle) setNextOffer(
	tx *vault.Txn,
	identity postingidentity.Identity,
	acceptancesNeeded int,
	answers postingAnswers,
) (bool, error) {
	if len(answers.acceptingPeers) >= acceptancesNeeded {
		return c.schedule.SetNextOfferAfterRedundancyMet(tx, identity)
	}

	return c.schedule.SetNextOfferAfterRedundancyMissed(tx, identity, answers.requestedPause)
}

func replicaRingFractionsOf(offers []postingoffer.PostingOffer, round offerRound) []float64 {
	var ringFractions []float64
	for _, offer := range offers {
		for _, peer := range round[postingidentity.IdentityOf(offer.Posting)].acceptingPeers {
			ringFractions = append(ringFractions, offer.PostingPosition.DistanceTo(
				yacymodel.DHTRingPositionOf(peer),
			).FractionOfDHTRing())
		}
	}

	return ringFractions
}

func (c *Cycle) reportBatchUnwritten(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "distribution batch not written", slog.Any("error", err))
	c.cycleObserver.ObserveBatchAborted(string(AbortBatchUnwritten))
}
