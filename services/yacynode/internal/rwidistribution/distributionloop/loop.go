// Package distributionloop keeps sending postings. It walks the ring by arc: the
// stretch of ring whose window peers are the same. A worker takes an arc whose
// window peers are free and reads only its due word partitions. A batch sends
// them to the window peers at once and writes down what happened. A new posting
// or a roster change wakes the workers; no worker polls.
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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionschedule"
)

const retryDelayAfterVaultError = 10 * time.Second

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
		currentHolders []yacymodel.Hash,
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
	PostingsPerBatch    int
	PostingsPerTransfer int
	AmountOfWorkers     int
	MinReachablePeers   int
}

type Loop struct {
	vault           *vault.Vault
	offers          *wordpartitionoffer.Offers
	undelivered     *undeliveredpostings.Postings
	schedule        *wordpartitionschedule.Schedule
	handoff         WordPartitionHandoff
	sender          offerSender
	roster          ReachablePeers
	clock           Clock
	batchObserver   BatchObserver
	dhtRingObserver DHTRingObserver
	walk            *arcWalk
	batches         arcBatches
	gauges          *gaugeObservation
	config          Config
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func New(
	v *vault.Vault,
	offers *wordpartitionoffer.Offers,
	undelivered *undeliveredpostings.Postings,
	schedule *wordpartitionschedule.Schedule,
	handoff WordPartitionHandoff,
	transfers PostingTransfers,
	eligibility PeerEligibility,
	roster ReachablePeers,
	clock Clock,
	batchObserver BatchObserver,
	backlogObserver BacklogObserver,
	dhtRingObserver DHTRingObserver,
	config Config,
) *Loop {
	walk := newArcWalk(clock, eligibility)

	return &Loop{
		vault:       v,
		offers:      offers,
		undelivered: undelivered,
		schedule:    schedule,
		handoff:     handoff,
		sender: offerSender{
			transfers:           transfers,
			eligibility:         eligibility,
			clock:               clock,
			observer:            batchObserver,
			postingsPerTransfer: config.PostingsPerTransfer,
		},
		roster:          roster,
		clock:           clock,
		batchObserver:   batchObserver,
		dhtRingObserver: dhtRingObserver,
		walk:            walk,
		batches: arcBatches{
			walk:     walk,
			schedule: schedule,
			reader: arcReader{
				vault:                  v,
				schedule:               schedule,
				amountOfWordPartitions: config.PostingsPerBatch,
			},
			offers:         offers,
			postingsBudget: config.PostingsPerBatch,
		},
		gauges: &gaugeObservation{
			vault:           v,
			undelivered:     undelivered,
			roster:          roster,
			clock:           clock,
			walk:            walk,
			backlogObserver: backlogObserver,
			dhtRingObserver: dhtRingObserver,
		},
		config: config,
	}
}

func (l *Loop) Run(ctx context.Context) {
	var workers sync.WaitGroup
	for range l.config.AmountOfWorkers {
		workers.Go(func() { l.offerBatchesUntilDone(ctx) })
	}
	l.gauges.observeUntilDone(ctx)
	workers.Wait()
}

func (l *Loop) ObservePostingAdmitted(position yacymodel.DHTRingPosition) {
	l.walk.signalAdmission(position)
}

func (l *Loop) ObserveReachablePeers(int) {
	l.walk.signalReachablePeers()
}

func (l *Loop) offerBatchesUntilDone(ctx context.Context) {
	for ctx.Err() == nil {
		l.offerNextBatch(ctx)
	}
}

func (l *Loop) offerNextBatch(ctx context.Context) {
	wake := l.walk.wakeSignal()
	peers := l.roster.ReachablePeers(ctx)
	if len(peers) < l.config.MinReachablePeers {
		waitOn(ctx, l.clock, wake, wakeAlarm{})

		return
	}
	search, err := l.batches.nextFrom(ctx, peers)
	if err != nil {
		l.reportDuePostingsUnread(ctx, err)
		sleepOn(ctx, l.clock, retryDelayAfterVaultError)

		return
	}
	if !search.hasBatch {
		waitOn(ctx, l.clock, wake, search.alarm)

		return
	}

	l.offerBatch(ctx, search.batch)
}

func (l *Loop) offerBatch(ctx context.Context, batch dueBatch) {
	answers := l.sender.send(ctx, batch.offers)
	if err := l.commitBatch(ctx, batch.offers, answers); err != nil {
		l.reportBatchUnwritten(ctx, err)
		l.walk.abandon(batch.arc)
		sleepOn(ctx, l.clock, retryDelayAfterVaultError)

		return
	}
	l.walk.complete(batch.arc)
	l.batchObserver.ObserveBatchCompleted()
}

func (l *Loop) reportDuePostingsUnread(ctx context.Context, err error) {
	slog.ErrorContext(ctx, "due postings not read", slog.Any("error", err))
	l.batchObserver.ObserveBatchAborted(string(AbortDuePostingsUnread))
}

func (l *Loop) commitBatch(
	ctx context.Context,
	offers []wordpartitionoffer.Offer,
	answers map[yacymodel.Hash]wordpartitionoffer.PeerAnswer,
) error {
	outcomes := outcomesFrom(offers, answers)
	if err := l.vault.Update(ctx, func(tx *vault.Txn) error {
		for _, outcome := range outcomes {
			if err := l.recordOutcome(ctx, tx, outcome); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	l.dhtRingObserver.ObserveHolderRingFractions(ringFractionsOf(outcomes))

	return nil
}

func outcomesFrom(
	offers []wordpartitionoffer.Offer,
	answers map[yacymodel.Hash]wordpartitionoffer.PeerAnswer,
) []wordpartitionoffer.Outcome {
	outcomes := make([]wordpartitionoffer.Outcome, 0, len(offers))
	for _, offer := range offers {
		outcomes = append(outcomes, offer.OutcomeFrom(answers))
	}

	return outcomes
}

func (l *Loop) recordOutcome(
	ctx context.Context,
	tx *vault.Txn,
	outcome wordpartitionoffer.Outcome,
) error {
	if err := l.recordHolders(tx, outcome); err != nil {
		return err
	}
	amountOfPostingsDelivered, err := l.markDelivered(tx, outcome.DeliveredPostings)
	if err != nil {
		return err
	}
	amountOfPostingsHandedOff, err := l.handoff.HandOffIfHeldByResponsiblePeers(
		ctx, tx, outcome.WordPartition, outcome.Holders, outcome.ResponsiblePeers,
	)
	if err != nil {
		return err
	}
	tx.RunAfterCommit(func() {
		l.batchObserver.ObserveStaleHoldersDropped(outcome.AmountOfStaleHolders)
		l.batchObserver.ObservePostingsHandedOff(amountOfPostingsHandedOff)
		l.batchObserver.ObservePostingsDelivered(amountOfPostingsDelivered)
	})

	return nil
}

func (l *Loop) recordHolders(tx *vault.Txn, outcome wordpartitionoffer.Outcome) error {
	switch {
	case outcome.IsAtRedundancy && outcome.IsRefresh:
		return l.schedule.RecordRefreshed(tx, outcome.WordPartition, outcome.Holders)
	case outcome.IsAtRedundancy:
		return l.schedule.RecordAtRedundancy(tx, outcome.WordPartition, outcome.Holders)
	}

	return l.schedule.RecordShortOfRedundancy(tx, outcome.WordPartition, outcome.Holders)
}

func (l *Loop) markDelivered(
	tx *vault.Txn,
	deliveredPostings []undeliveredpostings.Posting,
) (amountOfPostingsDelivered int, err error) {
	for _, posting := range deliveredPostings {
		wasMarked, err := l.undelivered.MarkDelivered(tx, posting)
		if err != nil {
			return 0, err
		}
		if wasMarked {
			amountOfPostingsDelivered++
		}
	}

	return amountOfPostingsDelivered, nil
}

func ringFractionsOf(outcomes []wordpartitionoffer.Outcome) []float64 {
	var ringFractions []float64
	for _, outcome := range outcomes {
		for _, acceptance := range outcome.Acceptances {
			ringFraction := outcome.Position.DistanceTo(
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
