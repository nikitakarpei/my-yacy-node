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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingofferinterval"
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
	HandOff(
		ctx context.Context,
		tx *vault.Txn,
		posting yacymodel.RWIPosting,
		holders []yacymodel.Hash,
	) (bool, error)
}

type Config struct {
	OfferInterval     postingofferinterval.Bounds
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
	transfers       *postingtransfer.PostingTransfers
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

	offers, gonePostings, err := c.postingOffers.DueNowIn(
		ctx, sector, c.config.PostingsPerBatch, peers,
	)
	if err != nil {
		c.reportDuePostingsUnread(ctx, err)

		return false
	}
	amountOfDuePostings := len(offers) + len(gonePostings)
	if amountOfDuePostings < c.config.PostingsPerBatch {
		c.sectorTurns.markDrained(sector)
	}
	if amountOfDuePostings == 0 {
		return true
	}

	return c.offerBatch(ctx, offers, gonePostings)
}

func (c *Cycle) reportDuePostingsUnread(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "due postings not read", slog.Any("error", err))
	c.cycleObserver.ObserveBatchAborted(string(AbortDuePostingsUnread))
}

func (c *Cycle) offerBatch(
	ctx context.Context,
	offers []postingoffer.PostingOffer,
	gonePostings []postingidentity.Identity,
) bool {
	c.reportPostingsGone(ctx, gonePostings)

	round := offerDuePostings(ctx, c.transfers, c.answers, offers)
	if err := c.commitBatch(ctx, offers, gonePostings, round); err != nil {
		c.reportBatchUnwritten(ctx, err)

		return false
	}

	return true
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
	gonePostings []postingidentity.Identity,
	round offerRound,
) error {
	err := c.vault.Update(ctx, func(tx *vault.Txn) error {
		acceptingPeers := acceptingPeersByPosting(round.acceptances)
		for _, offer := range offers {
			identity := postingidentity.IdentityOf(offer.Posting)
			if err := c.recordOfferOutcome(
				ctx, tx, offer, acceptingPeers[identity], round.pauses[identity],
			); err != nil {
				return err
			}
		}

		return c.forgetGonePostings(tx, gonePostings)
	})
	if err != nil {
		return err
	}

	c.dhtRingObserver.ObserveReplicaRingFractions(
		replicaRingFractionsOf(offers, round.acceptances),
	)

	return nil
}

func acceptingPeersByPosting(
	acceptances []peerAcceptance,
) map[postingidentity.Identity][]yacymodel.Hash {
	byPosting := make(map[postingidentity.Identity][]yacymodel.Hash)
	for _, acceptance := range acceptances {
		for _, posting := range acceptance.postings {
			identity := postingidentity.IdentityOf(posting)
			byPosting[identity] = append(byPosting[identity], acceptance.holder)
		}
	}

	return byPosting
}

func (c *Cycle) recordOfferOutcome(
	ctx context.Context,
	tx *vault.Txn,
	offer postingoffer.PostingOffer,
	acceptingPeers []yacymodel.Hash,
	requestedPause time.Duration,
) error {
	identity := postingidentity.IdentityOf(offer.Posting)
	recordedHolders, err := c.replicas.HoldersOf(tx, identity)
	if err != nil {
		return err
	}
	change := postingreplicas.HolderChange{
		StaleHolders:   offer.StaleHolders,
		AcceptingPeers: acceptingPeers,
	}
	holders := change.HoldersFrom(recordedHolders)

	wasHandedOff, err := c.handoff.HandOff(ctx, tx, offer.Posting, holders)
	if err != nil {
		return err
	}
	if wasHandedOff {
		c.observeStaleReplicasDropped(tx, change.AmountOfStaleHoldersIn(recordedHolders))
		c.observePostingHandedOff(tx)

		return nil
	}

	wasRescheduled, err := c.setNextOffer(tx, offer, acceptingPeers, requestedPause)
	if err != nil || !wasRescheduled {
		return err
	}
	c.observeStaleReplicasDropped(tx, change.AmountOfStaleHoldersIn(recordedHolders))

	return c.replicas.RecordHolders(tx, identity, recordedHolders, holders)
}

func (c *Cycle) observeStaleReplicasDropped(tx *vault.Txn, droppedReplicas int) {
	tx.RunAfterCommit(func() { c.cycleObserver.ObserveStaleReplicasDropped(droppedReplicas) })
}

func (c *Cycle) observePostingHandedOff(tx *vault.Txn) {
	tx.RunAfterCommit(func() { c.cycleObserver.ObservePostingsHandedOff(1) })
}

func (c *Cycle) setNextOffer(
	tx *vault.Txn,
	offer postingoffer.PostingOffer,
	acceptingPeers []yacymodel.Hash,
	requestedPause time.Duration,
) (bool, error) {
	identity := postingidentity.IdentityOf(offer.Posting)
	if len(acceptingPeers) >= offer.AcceptancesNeeded {
		return c.schedule.SetNextOfferAfterRedundancyMet(tx, identity, c.config.OfferInterval)
	}

	return c.schedule.SetNextOfferAfterRedundancyMissed(
		tx, identity, c.config.OfferInterval, requestedPause,
	)
}

func replicaRingFractionsOf(
	offers []postingoffer.PostingOffer,
	acceptances []peerAcceptance,
) []float64 {
	offerByPosting := make(map[postingidentity.Identity]postingoffer.PostingOffer, len(offers))
	for _, offer := range offers {
		identity := postingidentity.IdentityOf(offer.Posting)
		offerByPosting[identity] = offer
	}

	var ringFractions []float64
	for _, acceptance := range acceptances {
		for _, posting := range acceptance.postings {
			identity := postingidentity.IdentityOf(posting)
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

func (c *Cycle) forgetGonePostings(
	tx *vault.Txn,
	gonePostings []postingidentity.Identity,
) error {
	for _, identity := range gonePostings {
		gonePosting := yacymodel.RWIPosting{WordHash: identity.Word, URLHash: identity.URL}
		if err := c.schedule.PostingPurged(tx, gonePosting); err != nil {
			return err
		}
		if err := c.replicas.PostingPurged(tx, gonePosting); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cycle) reportBatchUnwritten(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "distribution batch not written", slog.Any("error", err))
	c.cycleObserver.ObserveBatchAborted(string(AbortBatchUnwritten))
}
