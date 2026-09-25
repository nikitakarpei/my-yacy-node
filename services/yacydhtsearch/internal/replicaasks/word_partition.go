package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type placedAsk[Ask any] struct {
	ask                 Ask
	replica             Replica
	placeInReplicaOrder int
}

type wordPartition[Ask any, Answered any] struct {
	replicaAsks              Asks[Ask, Answered]
	chosenPeers              *chosenPeers
	asksInReplicaOrder       []placedAsk[Ask]
	asksLeft                 []placedAsk[Ask]
	firstAsks                []placedAsk[Ask]
	calls                    []*replicaCall[Ask, Answered]
	hedgesDue                chan *replicaCall[Ask, Answered]
	callOutcomes             chan replicaCallOutcome[Ask, Answered]
	settledBy                SettledBy
	coveringAskPutOn         PutOn
	amountOfCallsOutstanding int
	amountOfCoveringAnswers  int
}

type replicaCall[Ask any, Answered any] struct {
	placedAsk  placedAsk[Ask]
	putOn      PutOn
	ended      bool
	hedgeTimer *time.Timer
	answer     yacymodel.Optional[Answered]
	coverage   Coverage
}

type replicaCallOutcome[Ask any, Answered any] struct {
	call     *replicaCall[Ask, Answered]
	answered bool
	answer   Answered
	coverage Coverage
}

func wordPartitionOf[Ask any, Answered any](
	asksInReplicaOrder []placedAsk[Ask],
	replicaAsks Asks[Ask, Answered],
	chosenPeers *chosenPeers,
) *wordPartition[Ask, Answered] {
	return &wordPartition[Ask, Answered]{
		replicaAsks:        replicaAsks,
		chosenPeers:        chosenPeers,
		asksInReplicaOrder: asksInReplicaOrder,
		asksLeft:           asksInReplicaOrder,
		hedgesDue:          make(chan *replicaCall[Ask, Answered], len(asksInReplicaOrder)),
		callOutcomes:       make(chan replicaCallOutcome[Ask, Answered], len(asksInReplicaOrder)),
	}
}

func (partition *wordPartition[Ask, Answered]) chooseTheFirstAsks() {
	for range partition.replicaAsks.amountOfReplicasCoveringAPartition {
		placedAsk, chosen := partition.chooseTheNextAsk()
		if !chosen {
			return
		}
		partition.firstAsks = append(partition.firstAsks, placedAsk)
	}
}

func (partition *wordPartition[Ask, Answered]) chooseTheNextAsk() (placedAsk[Ask], bool) {
	for !partition.noAskIsLeft() {
		placedAsk := partition.asksLeft[0]
		partition.asksLeft = partition.asksLeft[1:]
		if partition.chosenPeers.choose(placedAsk.replica.Peer.Hash) {
			return placedAsk, true
		}
	}

	return placedAsk[Ask]{}, false
}

func (partition *wordPartition[Ask, Answered]) noAskIsLeft() bool {
	return len(partition.asksLeft) == 0
}

func (partition *wordPartition[Ask, Answered]) askUntilSettled(ctx context.Context) {
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	for _, placedAsk := range partition.firstAsks {
		partition.putTheAsk(askingContext, placedAsk, PutOnStart)
	}
	partition.settleWhenNothingIsLeftToAsk()
	for !partition.isSettled() {
		partition.takeTheNextEvent(askingContext)
	}
	partition.stopTheHedgeTimers()
}

func (partition *wordPartition[Ask, Answered]) putTheAsk(
	ctx context.Context,
	placedAsk placedAsk[Ask],
	putOn PutOn,
) {
	call := &replicaCall[Ask, Answered]{
		placedAsk: placedAsk,
		putOn:     putOn,
		answer:    yacymodel.None[Answered](),
	}
	call.hedgeTimer = time.AfterFunc(
		partition.replicaAsks.hedgeDelay.HedgeDelayOf(ctx, placedAsk.replica.Peer),
		func() { partition.hedgesDue <- call },
	)
	partition.calls = append(partition.calls, call)
	partition.amountOfCallsOutstanding++
	go partition.callTheReplica(ctx, call, placedAsk.ask)
}

func (partition *wordPartition[Ask, Answered]) callTheReplica(
	ctx context.Context,
	call *replicaCall[Ask, Answered],
	ask Ask,
) {
	answer, answered := partition.replicaAsks.replicaCalls.AnswerTo(ctx, ask)
	if !answered {
		partition.callOutcomes <- replicaCallOutcome[Ask, Answered]{call: call}

		return
	}
	partition.callOutcomes <- replicaCallOutcome[Ask, Answered]{
		call:     call,
		answered: true,
		answer:   answer,
		coverage: partition.replicaAsks.replicaCalls.CoverageFrom(answer),
	}
}

func (partition *wordPartition[Ask, Answered]) settleWhenNothingIsLeftToAsk() {
	if !partition.isSettled() && partition.amountOfCallsOutstanding == 0 &&
		partition.noAskIsLeft() {
		partition.settledBy = SettledByNoReplicaLeft
	}
}

func (partition *wordPartition[Ask, Answered]) isSettled() bool {
	return partition.settledBy != ""
}

func (partition *wordPartition[Ask, Answered]) takeTheNextEvent(ctx context.Context) {
	if ctx.Err() != nil {
		partition.settledBy = SettledByDeadline

		return
	}
	select {
	case call := <-partition.hedgesDue:
		partition.takeTheHedgeDue(ctx, call)
	case outcome := <-partition.callOutcomes:
		partition.takeTheCallOutcome(ctx, outcome)
	case <-ctx.Done():
		partition.settledBy = SettledByDeadline
	}
}

func (partition *wordPartition[Ask, Answered]) takeTheHedgeDue(
	ctx context.Context,
	call *replicaCall[Ask, Answered],
) {
	if call.ended {
		return
	}
	partition.askTheNextReplica(ctx, PutOnHedgeDelay)
}

func (partition *wordPartition[Ask, Answered]) askTheNextReplica(ctx context.Context, putOn PutOn) {
	placedAsk, chosen := partition.chooseTheNextAsk()
	if !chosen {
		return
	}
	partition.putTheAsk(ctx, placedAsk, putOn)
}

func (partition *wordPartition[Ask, Answered]) takeTheCallOutcome(
	ctx context.Context,
	outcome replicaCallOutcome[Ask, Answered],
) {
	partition.endTheCall(outcome)
	partition.recordTheAnswer(outcome)
	partition.askTheNextReplicaWhenNotCovering(ctx, outcome)
	partition.countTheCoveringAnswer(outcome)
	partition.settleWhenNothingIsLeftToAsk()
}

func (partition *wordPartition[Ask, Answered]) endTheCall(
	outcome replicaCallOutcome[Ask, Answered],
) {
	partition.amountOfCallsOutstanding--
	outcome.call.ended = true
}

func (partition *wordPartition[Ask, Answered]) recordTheAnswer(
	outcome replicaCallOutcome[Ask, Answered],
) {
	if outcome.answered {
		outcome.call.answer = yacymodel.Some(outcome.answer)
		outcome.call.coverage = outcome.coverage
	}
}

func (partition *wordPartition[Ask, Answered]) askTheNextReplicaWhenNotCovering(
	ctx context.Context,
	outcome replicaCallOutcome[Ask, Answered],
) {
	switch {
	case !outcome.answered:
		partition.askTheNextReplica(ctx, PutOnFailure)
	case !outcome.coverage.coversThePartition():
		partition.askTheNextReplica(ctx, PutOnEmptyAnswer)
	}
}

func (partition *wordPartition[Ask, Answered]) countTheCoveringAnswer(
	outcome replicaCallOutcome[Ask, Answered],
) {
	if !outcome.coverage.coversThePartition() {
		return
	}
	partition.amountOfCoveringAnswers++
	if partition.amountOfCoveringAnswers ==
		partition.replicaAsks.amountOfReplicasCoveringAPartition {
		partition.settledBy = SettledByCoverage
		partition.coveringAskPutOn = outcome.call.putOn
	}
}

func (partition *wordPartition[Ask, Answered]) stopTheHedgeTimers() {
	for _, call := range partition.calls {
		call.hedgeTimer.Stop()
	}
}

func (
	partition *wordPartition[Ask, Answered],
) settledWordPartition() SettledWordPartition[Ask, Answered] {
	callOfEachPlace := make(map[int]*replicaCall[Ask, Answered], len(partition.calls))
	for _, call := range partition.calls {
		callOfEachPlace[call.placedAsk.placeInReplicaOrder] = call
	}
	askOutcomes := make(peerasks.AskOutcomes[Ask, Answered], 0, len(partition.asksInReplicaOrder))
	for _, placedAsk := range partition.asksInReplicaOrder {
		askOutcome := peerasks.AskOutcome[Ask, Answered]{Ask: placedAsk.ask}
		if call, put := callOfEachPlace[placedAsk.placeInReplicaOrder]; put {
			askOutcome.Put = true
			askOutcome.Answer = call.answer
		}
		askOutcomes = append(askOutcomes, askOutcome)
	}

	return SettledWordPartition[Ask, Answered]{AskOutcomes: askOutcomes}
}

func (partition *wordPartition[Ask, Answered]) performedWordPartition() PerformedWordPartition {
	return PerformedWordPartition{
		SettledBy:               partition.settledBy,
		CoveringAskPutOn:        partition.coveringAskPutOn,
		AmountOfDocumentsListed: partition.amountOfDocumentsListed(),
		AsksPutOn:               partition.asksPutOn(),
	}
}

func (partition *wordPartition[Ask, Answered]) amountOfDocumentsListed() int {
	amountOfDocumentsListed := 0
	for _, call := range partition.calls {
		amountOfDocumentsListed += call.coverage.AmountOfDocumentsListed
	}

	return amountOfDocumentsListed
}

func (partition *wordPartition[Ask, Answered]) asksPutOn() []PutOn {
	asksPutOn := make([]PutOn, 0, len(partition.calls))
	for _, call := range partition.calls {
		asksPutOn = append(asksPutOn, call.putOn)
	}

	return asksPutOn
}
