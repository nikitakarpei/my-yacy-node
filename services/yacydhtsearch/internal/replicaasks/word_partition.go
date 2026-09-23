package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type placedAsk[Ask any] struct {
	ask           Ask
	placeInTheRun int
}

type placedAskOutcome[Ask any, Answered any] struct {
	placeInTheRun int
	outcome       peerasks.AskOutcome[Ask, Answered]
}

type wordPartition[Ask any, Answered any] struct {
	kind                               askKind[Ask, Answered]
	amountOfReplicasCoveringAPartition int
	chosenPeers                        *chosenPeers
	asksLeft                           []placedAsk[Ask]
	firstAsks                          []placedAsk[Ask]
	calls                              []*replicaCall[Ask, Answered]
	hedgesDue                          chan *replicaCall[Ask, Answered]
	callOutcomes                       chan replicaCallOutcome[Ask, Answered]
	settledBy                          SettledBy
	coveringAskPutOn                   PutOn
	amountOfCallsOutstanding           int
	amountOfListingAnswers             int
}

type replicaCall[Ask any, Answered any] struct {
	placedAsk  placedAsk[Ask]
	putOn      PutOn
	ended      bool
	hedgeTimer *time.Timer
	answer     yacymodel.Optional[Answered]
}

type replicaCallOutcome[Ask any, Answered any] struct {
	call           *replicaCall[Ask, Answered]
	answered       bool
	listsDocuments bool
	answer         Answered
}

func wordPartitionOf[Ask any, Answered any](
	asksInReplicaOrder []placedAsk[Ask],
	kind askKind[Ask, Answered],
	amountOfReplicasCoveringAPartition int,
	chosenPeers *chosenPeers,
) *wordPartition[Ask, Answered] {
	return &wordPartition[Ask, Answered]{
		kind:                               kind,
		amountOfReplicasCoveringAPartition: amountOfReplicasCoveringAPartition,
		chosenPeers:                        chosenPeers,
		asksLeft:                           asksInReplicaOrder,
		hedgesDue: make(
			chan *replicaCall[Ask, Answered],
			len(asksInReplicaOrder),
		),
		callOutcomes: make(
			chan replicaCallOutcome[Ask, Answered],
			len(asksInReplicaOrder),
		),
	}
}

func (partition *wordPartition[Ask, Answered]) chooseTheFirstAsks() {
	for range partition.amountOfReplicasCoveringAPartition {
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
		if partition.chosenPeers.choose(partition.peerOf(placedAsk)) {
			return placedAsk, true
		}
	}

	return placedAsk[Ask]{}, false
}

func (partition *wordPartition[Ask, Answered]) noAskIsLeft() bool {
	return len(partition.asksLeft) == 0
}

func (partition *wordPartition[Ask, Answered]) peerOf(placedAsk placedAsk[Ask]) yacymodel.Hash {
	return partition.kind.peerOf(placedAsk.ask)
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
		partition.kind.hedgeDelayOf(ctx, placedAsk.ask),
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
	answer, answered := partition.kind.putAsk(ctx, ask)
	partition.callOutcomes <- replicaCallOutcome[Ask, Answered]{
		call:           call,
		answered:       answered,
		answer:         answer,
		listsDocuments: answered && partition.kind.amountOfDocumentsListedIn(answer) > 0,
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

func (partition *wordPartition[Ask, Answered]) askTheNextReplica(
	ctx context.Context,
	putOn PutOn,
) {
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
	partition.askTheNextReplicaWhenNotListing(ctx, outcome)
	partition.countTheListingAnswer(outcome)
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
	}
}

func (partition *wordPartition[Ask, Answered]) askTheNextReplicaWhenNotListing(
	ctx context.Context,
	outcome replicaCallOutcome[Ask, Answered],
) {
	switch {
	case !outcome.answered:
		partition.askTheNextReplica(ctx, PutOnFailure)
	case !outcome.listsDocuments:
		partition.askTheNextReplica(ctx, PutOnEmptyAnswer)
	}
}

func (partition *wordPartition[Ask, Answered]) countTheListingAnswer(
	outcome replicaCallOutcome[Ask, Answered],
) {
	if !outcome.listsDocuments {
		return
	}
	partition.amountOfListingAnswers++
	if partition.amountOfListingAnswers == partition.amountOfReplicasCoveringAPartition {
		partition.settledBy = SettledByCoverage
		partition.coveringAskPutOn = outcome.call.putOn
	}
}

func (partition *wordPartition[Ask, Answered]) stopTheHedgeTimers() {
	for _, call := range partition.calls {
		call.hedgeTimer.Stop()
	}
}

func (partition *wordPartition[Ask, Answered]) settledWordPartition() SettledWordPartition {
	return SettledWordPartition{
		SettledBy:               partition.settledBy,
		CoveringAskPutOn:        partition.coveringAskPutOn,
		AmountOfDocumentsListed: partition.amountOfDocumentsListed(),
		AsksPutOn:               partition.asksPutOn(),
	}
}

func (partition *wordPartition[Ask, Answered]) amountOfDocumentsListed() int {
	amountOfDocumentsListed := 0
	for _, call := range partition.calls {
		answer, answered := call.answer.Get()
		if !answered {
			continue
		}
		amountOfDocumentsListed += partition.kind.amountOfDocumentsListedIn(answer)
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

func (partition *wordPartition[Ask, Answered]) placedAskOutcomes() []placedAskOutcome[Ask, Answered] {
	placedAskOutcomes := make([]placedAskOutcome[Ask, Answered], 0, len(partition.calls))
	for _, call := range partition.calls {
		placedAskOutcomes = append(placedAskOutcomes, placedAskOutcome[Ask, Answered]{
			placeInTheRun: call.placedAsk.placeInTheRun,
			outcome: peerasks.AskOutcome[Ask, Answered]{
				Ask:    call.placedAsk.ask,
				Put:    true,
				Answer: call.answer,
			},
		})
	}

	return placedAskOutcomes
}
