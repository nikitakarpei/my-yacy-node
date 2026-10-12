// Package distributionloop keeps sending postings: it asks wordpartitionoffer
// which word partitions are due now and what each peer gets, sends the
// postings, and writes down what happened. Workers take the ring sectors in
// turn, one batch at a time, each in a sector no other batch holds. After a
// pass, a sector waits the rescan delay; a batch sends to all its peers at once.
package distributionloop

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
	SectorRescanDelay time.Duration
	MinReachablePeers int
}

type Loop struct {
	vault           *vault.Vault
	offers          *wordpartitionoffer.WordPartitionOffers
	undelivered     *undeliveredpostings.UndeliveredPostings
	holders         *wordpartitionholders.Holders
	handoff         WordPartitionHandoff
	transfers       PostingTransfers
	cooldowns       Cooldowns
	roster          ReachablePeers
	clock           Clock
	batchObserver   BatchObserver
	dhtRingObserver DHTRingObserver
	sectorTurns     *sectorTurns
	gauges          *gaugeObservation
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
	clock Clock,
	batchObserver BatchObserver,
	backlogObserver BacklogObserver,
	dhtRingObserver DHTRingObserver,
	config Config,
) *Loop {
	return &Loop{
		vault:           v,
		offers:          offers,
		undelivered:     undelivered,
		holders:         holders,
		handoff:         handoff,
		transfers:       transfers,
		cooldowns:       cooldowns,
		roster:          roster,
		clock:           clock,
		batchObserver:   batchObserver,
		dhtRingObserver: dhtRingObserver,
		sectorTurns:     newSectorTurns(clock, config.SectorRescanDelay),
		gauges: &gaugeObservation{
			vault:           v,
			offers:          offers,
			undelivered:     undelivered,
			holders:         holders,
			roster:          roster,
			clock:           clock,
			backlogObserver: backlogObserver,
		},
		config: config,
	}
}

func (l *Loop) Run(ctx context.Context) {
	var activities sync.WaitGroup
	activities.Go(func() { l.gauges.observeUntilDone(ctx) })
	for range l.config.BatchConcurrency {
		activities.Go(func() { l.offerBatchesUntilDone(ctx) })
	}
	activities.Wait()
}

func (l *Loop) offerBatchesUntilDone(ctx context.Context) {
	for ctx.Err() == nil {
		l.offerNextBatch(ctx)
	}
}

func (l *Loop) offerNextBatch(ctx context.Context) {
	peers := l.roster.ReachablePeers(ctx)
	if len(peers) < l.config.MinReachablePeers {
		l.skipTooFewReachablePeers(ctx, len(peers))
		sleepOn(ctx, l.clock, l.config.SectorRescanDelay)

		return
	}
	sector, found := l.sectorTurns.claimNext()
	if !found {
		sleepOn(ctx, l.clock, l.sectorTurns.shortestRescanDelayLeft())

		return
	}
	defer l.sectorTurns.release(sector)

	l.offerDueBatchIn(ctx, sector, peers)
}

func (l *Loop) skipTooFewReachablePeers(ctx context.Context, reachablePeers int) {
	slog.DebugContext(
		ctx,
		"distribution batch skipped: too few reachable peers",
		slog.Int("reachablePeers", reachablePeers),
		slog.Int("minReachablePeers", l.config.MinReachablePeers),
	)
	l.batchObserver.ObserveBatchSkipped(string(SkipTooFewReachablePeers))
}

func (l *Loop) offerDueBatchIn(
	ctx context.Context,
	sector yacymodel.DHTRingSector,
	peers []yacymodel.Seed,
) {
	offers, next, err := l.offers.DueNowIn(ctx, sector, l.sectorTurns.cursorOf(sector), peers)
	if err != nil {
		l.reportDuePostingsUnread(ctx, err)
		l.sectorTurns.delayRescan(sector)

		return
	}
	if len(offers) > 0 {
		answers := sendOffers(ctx, l.transfers, l.cooldowns, offers, l.config.PostingsPerBatch)
		if err := l.commitBatch(ctx, offers, answers); err != nil {
			l.reportBatchUnwritten(ctx, err)
			l.sectorTurns.delayRescan(sector)

			return
		}
	}
	l.sectorTurns.advance(sector, next)
	if endsSectorPass(next) {
		l.sectorTurns.delayRescan(sector)
	}
	l.batchObserver.ObserveBatchCompleted()
}

func (l *Loop) reportDuePostingsUnread(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "due postings not read", slog.Any("error", err))
	l.batchObserver.ObserveBatchAborted(string(AbortDuePostingsUnread))
}

func (l *Loop) commitBatch(
	ctx context.Context,
	offers []wordpartitionoffer.WordPartitionOffer,
	answers map[yacymodel.Hash]wordpartitionoffer.PeerAnswer,
) error {
	outcomes := make([]wordpartitionoffer.Outcome, 0, len(offers))
	for _, offer := range offers {
		outcomes = append(outcomes, offer.OutcomeFrom(answers))
	}
	if err := l.vault.Update(ctx, func(tx *vault.Txn) error {
		for index, offer := range offers {
			if err := l.recordOutcome(ctx, tx, offer, outcomes[index]); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	l.dhtRingObserver.ObserveReplicaRingFractions(ringFractionsOf(offers, outcomes))

	return nil
}

func (l *Loop) recordOutcome(
	ctx context.Context,
	tx *vault.Txn,
	offer wordpartitionoffer.WordPartitionOffer,
	outcome wordpartitionoffer.Outcome,
) error {
	if outcome.HoldersChanged || offer.IsRefresh {
		if err := l.recordHolders(tx, offer.WordPartition, outcome); err != nil {
			return err
		}
	}
	if err := l.markDelivered(tx, outcome.DeliveredPostings); err != nil {
		return err
	}
	amountOfPostingsHandedOff, err := l.handoff.HandOffIfHeldByResponsiblePeers(
		ctx, tx, offer.WordPartition, outcome.Holders, offer.ResponsiblePeers,
	)
	if err != nil {
		return err
	}
	tx.RunAfterCommit(func() {
		l.batchObserver.ObserveStaleReplicasDropped(outcome.AmountOfDroppedHolders)
		l.batchObserver.ObservePostingsHandedOff(amountOfPostingsHandedOff)
	})

	return nil
}

func (l *Loop) recordHolders(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	outcome wordpartitionoffer.Outcome,
) error {
	if outcome.IsAtRedundancy {
		return l.holders.RecordAtRedundancy(tx, wordPartition, outcome.Holders)
	}

	return l.holders.RecordShortOfRedundancy(tx, wordPartition, outcome.Holders)
}

func (l *Loop) markDelivered(
	tx *vault.Txn,
	deliveredPostings []undeliveredpostings.UndeliveredPosting,
) error {
	for _, posting := range deliveredPostings {
		if _, err := l.undelivered.MarkDelivered(tx, posting); err != nil {
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

func (l *Loop) reportBatchUnwritten(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "distribution batch not written", slog.Any("error", err))
	l.batchObserver.ObserveBatchAborted(string(AbortBatchUnwritten))
}

func endsSectorPass(next wordpartitionoffer.Cursor) bool {
	return next == (wordpartitionoffer.Cursor{})
}
