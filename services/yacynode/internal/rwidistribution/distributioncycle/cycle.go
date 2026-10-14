// Package distributioncycle runs one distribution pass: it asks
// wordpartitionoffer which word partitions are due now and what each peer
// gets, sends the postings, and writes down what happened. Batches take the
// ring sectors in turn, several at once, each in a sector no other batch in
// flight holds; a batch sends to all its peers at once.
package distributioncycle

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/postingtransfer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionholders"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
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

type WordPartitionHandoff interface {
	HandOffIfHeldByResponsiblePeers(
		ctx context.Context,
		tx *vault.Txn,
		wordPartition yacymodel.WordPartition,
		holders []yacymodel.Hash,
		responsiblePeers []yacymodel.Hash,
	) (int, error)
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
	offers          *wordpartitionoffer.WordPartitionOffers
	undelivered     *undeliveredpostings.UndeliveredPostings
	holders         *wordpartitionholders.Holders
	handoff         WordPartitionHandoff
	transfers       PostingTransfers
	cooldowns       Cooldowns
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
	offers *wordpartitionoffer.WordPartitionOffers,
	undelivered *undeliveredpostings.UndeliveredPostings,
	holders *wordpartitionholders.Holders,
	handoff WordPartitionHandoff,
	transfers PostingTransfers,
	cooldowns Cooldowns,
	roster ReachablePeers,
	now func() time.Time,
	cycleObserver CycleObserver,
	dhtRingObserver DHTRingObserver,
	config Config,
) *Cycle {
	return &Cycle{
		vault:           v,
		offers:          offers,
		undelivered:     undelivered,
		holders:         holders,
		handoff:         handoff,
		transfers:       transfers,
		cooldowns:       cooldowns,
		roster:          roster,
		now:             now,
		cycleObserver:   cycleObserver,
		dhtRingObserver: dhtRingObserver,
		sectorTurns:     newSectorTurns(),
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

	c.offers.ReportPeersAcceptingRemoteIndex(peers)
	c.drainDueOffers(ctx, peers)
	c.cycleObserver.ObserveCycleCompleted()
}

func (c *Cycle) observeBacklog(ctx context.Context) {
	backlog, err := c.backlogNow(ctx)
	if err != nil {
		slog.WarnContext(ctx, "distribution backlog not observed", slog.Any("error", err))

		return
	}
	c.cycleObserver.ObserveUndeliveredPostings(backlog.amountOfUndeliveredPostings)
	c.cycleObserver.ObserveLongestRefreshLateness(backlog.longestRefreshLateness)
}

type backlog struct {
	amountOfUndeliveredPostings int
	longestRefreshLateness      time.Duration
}

func (c *Cycle) backlogNow(ctx context.Context) (backlog, error) {
	var current backlog
	err := c.vault.View(ctx, func(tx *vault.Txn) error {
		var err error
		current.amountOfUndeliveredPostings, err = c.undelivered.AmountOfPostings(tx)
		if err != nil {
			return err
		}
		current.longestRefreshLateness, err = c.longestRefreshLatenessIn(tx)

		return err
	})

	return current, err
}

func (c *Cycle) longestRefreshLatenessIn(tx *vault.Txn) (time.Duration, error) {
	earliestRefreshAt, found, err := c.holders.EarliestRefreshAt(tx)
	if err != nil || !found {
		return 0, err
	}

	return max(c.now().Sub(earliestRefreshAt), 0), nil
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

func (c *Cycle) drainDueOffers(ctx context.Context, peers []yacymodel.Seed) {
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

	offers, next, err := c.offers.DueNowIn(ctx, sector, c.sectorTurns.cursorOf(sector), peers)
	if err != nil {
		c.reportDuePostingsUnread(ctx, err)

		return false
	}
	if c.drainedBy(offers, next) {
		c.sectorTurns.markDrained(sector)
	}
	if len(offers) > 0 {
		answers := sendOffers(ctx, c.transfers, c.cooldowns, offers, c.config.PostingsPerBatch)
		if err := c.commitBatch(ctx, offers, answers); err != nil {
			c.reportBatchUnwritten(ctx, err)

			return false
		}
	}
	c.sectorTurns.advance(sector, next)

	return true
}

func (c *Cycle) reportDuePostingsUnread(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "due postings not read", slog.Any("error", err))
	c.cycleObserver.ObserveBatchAborted(string(AbortDuePostingsUnread))
}

func (c *Cycle) drainedBy(
	offers []wordpartitionoffer.WordPartitionOffer,
	next wordpartitionoffer.Cursor,
) bool {
	amountOfPostings := 0
	for _, offer := range offers {
		for _, recipient := range offer.Recipients {
			amountOfPostings += len(recipient.Postings)
		}
	}

	return next == (wordpartitionoffer.Cursor{}) || amountOfPostings < c.config.PostingsPerBatch
}

func (c *Cycle) commitBatch(
	ctx context.Context,
	offers []wordpartitionoffer.WordPartitionOffer,
	answers map[yacymodel.Hash]wordpartitionoffer.PeerAnswer,
) error {
	outcomes := make([]wordpartitionoffer.Outcome, 0, len(offers))
	for _, offer := range offers {
		outcomes = append(outcomes, offer.OutcomeFrom(answers))
	}
	if err := c.vault.Update(ctx, func(tx *vault.Txn) error {
		for index, offer := range offers {
			if err := c.recordOutcome(ctx, tx, offer, outcomes[index]); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	c.dhtRingObserver.ObserveReplicaRingFractions(ringFractionsOf(offers, outcomes))

	return nil
}

func (c *Cycle) recordOutcome(
	ctx context.Context,
	tx *vault.Txn,
	offer wordpartitionoffer.WordPartitionOffer,
	outcome wordpartitionoffer.Outcome,
) error {
	if outcome.HoldersChanged || offer.IsRefresh {
		if err := c.recordHolders(tx, offer.WordPartition, outcome); err != nil {
			return err
		}
	}
	if err := c.markDelivered(tx, outcome.DeliveredPostings); err != nil {
		return err
	}
	amountOfPostingsHandedOff, err := c.handoff.HandOffIfHeldByResponsiblePeers(
		ctx, tx, offer.WordPartition, outcome.Holders, offer.ResponsiblePeers,
	)
	if err != nil {
		return err
	}
	tx.RunAfterCommit(func() {
		c.cycleObserver.ObserveStaleReplicasDropped(outcome.AmountOfDroppedHolders)
		c.cycleObserver.ObservePostingsHandedOff(amountOfPostingsHandedOff)
	})

	return nil
}

func (c *Cycle) recordHolders(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	outcome wordpartitionoffer.Outcome,
) error {
	if outcome.IsAtRedundancy {
		return c.holders.RecordAtRedundancy(tx, wordPartition, outcome.Holders)
	}

	return c.holders.RecordShortOfRedundancy(tx, wordPartition, outcome.Holders)
}

func (c *Cycle) markDelivered(
	tx *vault.Txn,
	deliveredPostings []undeliveredpostings.UndeliveredPosting,
) error {
	for _, posting := range deliveredPostings {
		if _, err := c.undelivered.MarkDelivered(tx, posting); err != nil {
			return err
		}
	}

	return nil
}

func ringFractionsOf(
	offers []wordpartitionoffer.WordPartitionOffer,
	outcomes []wordpartitionoffer.Outcome,
) []float64 {
	var ringFractions []float64
	for index, offer := range offers {
		for _, acceptance := range outcomes[index].Acceptances {
			ringFraction := offer.Position.DistanceTo(
				yacymodel.DHTRingPositionOf(acceptance.Peer),
			).FractionOfDHTRing()
			for range acceptance.AmountOfPostings {
				ringFractions = append(ringFractions, ringFraction)
			}
		}
	}

	return ringFractions
}

func (c *Cycle) reportBatchUnwritten(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "distribution batch not written", slog.Any("error", err))
	c.cycleObserver.ObserveBatchAborted(string(AbortBatchUnwritten))
}
