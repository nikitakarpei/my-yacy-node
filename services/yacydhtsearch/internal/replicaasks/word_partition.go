package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartition[Ask any, Answered any] struct {
	asksInReplicaOrder                 []placedAsk[Ask]
	kind                               askKind[Ask, Answered]
	amountOfReplicasCoveringAPartition int
	chosenPeers                        *chosenPeers
}

type placedAsk[Ask any] struct {
	ask           Ask
	placeInTheRun int
}

type askedWordPartition[Ask any, Answered any] struct {
	settledWordPartition SettledWordPartition
	placedAskOutcomes    []placedAskOutcome[Ask, Answered]
}

type placedAskOutcome[Ask any, Answered any] struct {
	placeInTheRun int
	outcome       peerasks.AskOutcome[Ask, Answered]
}

type wordPartitionAsking[Ask any, Answered any] struct {
	partition                wordPartition[Ask, Answered]
	asksLeft                 []placedAsk[Ask]
	firstAsks                []placedAsk[Ask]
	calls                    []*replicaCall[Ask, Answered]
	hedgesDue                chan *replicaCall[Ask, Answered]
	callOutcomes             chan replicaCallOutcome[Ask, Answered]
	settledBy                SettledBy
	coveringAskPutOn         PutOn
	amountOfCallsOutstanding int
	amountOfListingAnswers   int
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

func (partition wordPartition[Ask, Answered]) asking() *wordPartitionAsking[Ask, Answered] {
	return &wordPartitionAsking[Ask, Answered]{
		partition: partition,
		asksLeft:  partition.asksInReplicaOrder,
		hedgesDue: make(chan *replicaCall[Ask, Answered], len(partition.asksInReplicaOrder)),
		callOutcomes: make(
			chan replicaCallOutcome[Ask, Answered],
			len(partition.asksInReplicaOrder),
		),
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) chooseTheFirstAsks() {
	for range asking.partition.amountOfReplicasCoveringAPartition {
		placedAsk, chosen := asking.chooseTheNextAsk()
		if !chosen {
			return
		}
		asking.firstAsks = append(asking.firstAsks, placedAsk)
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) chooseTheNextAsk() (placedAsk[Ask], bool) {
	for !asking.noAskIsLeft() {
		placedAsk := asking.asksLeft[0]
		asking.asksLeft = asking.asksLeft[1:]
		if asking.partition.chosenPeers.choose(asking.partition.peerOf(placedAsk)) {
			return placedAsk, true
		}
	}

	return placedAsk[Ask]{}, false
}

func (asking *wordPartitionAsking[Ask, Answered]) noAskIsLeft() bool {
	return len(asking.asksLeft) == 0
}

func (partition wordPartition[Ask, Answered]) peerOf(placedAsk placedAsk[Ask]) yacymodel.Hash {
	return partition.kind.peerOf(placedAsk.ask)
}

func (asking *wordPartitionAsking[Ask, Answered]) askUntilSettled(
	ctx context.Context,
) askedWordPartition[Ask, Answered] {
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	for _, placedAsk := range asking.firstAsks {
		asking.putTheAsk(askingContext, placedAsk, PutOnStart)
	}
	asking.settleWhenNothingIsLeftToAsk()
	for !asking.isSettled() {
		asking.takeTheNextEvent(askingContext)
	}
	asking.stopTheHedgeTimers()

	return asking.askedWordPartition()
}

func (asking *wordPartitionAsking[Ask, Answered]) putTheAsk(
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
		asking.partition.kind.hedgeDelayOf(ctx, placedAsk.ask),
		func() { asking.hedgesDue <- call },
	)
	asking.calls = append(asking.calls, call)
	asking.amountOfCallsOutstanding++
	go asking.callTheReplica(ctx, call, placedAsk.ask)
}

func (asking *wordPartitionAsking[Ask, Answered]) callTheReplica(
	ctx context.Context,
	call *replicaCall[Ask, Answered],
	ask Ask,
) {
	answer, answered := asking.partition.kind.putAsk(ctx, ask)
	asking.callOutcomes <- replicaCallOutcome[Ask, Answered]{
		call:           call,
		answered:       answered,
		answer:         answer,
		listsDocuments: answered && asking.partition.kind.amountOfDocumentsListedIn(answer) > 0,
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) settleWhenNothingIsLeftToAsk() {
	if !asking.isSettled() && asking.amountOfCallsOutstanding == 0 && asking.noAskIsLeft() {
		asking.settledBy = SettledByNoReplicaLeft
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) isSettled() bool {
	return asking.settledBy != ""
}

func (asking *wordPartitionAsking[Ask, Answered]) takeTheNextEvent(ctx context.Context) {
	if ctx.Err() != nil {
		asking.settledBy = SettledByDeadline

		return
	}
	select {
	case call := <-asking.hedgesDue:
		asking.takeTheHedgeDue(ctx, call)
	case outcome := <-asking.callOutcomes:
		asking.takeTheCallOutcome(ctx, outcome)
	case <-ctx.Done():
		asking.settledBy = SettledByDeadline
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) takeTheHedgeDue(
	ctx context.Context,
	call *replicaCall[Ask, Answered],
) {
	if call.ended {
		return
	}
	asking.askTheNextReplica(ctx, PutOnHedgeDelay)
}

func (asking *wordPartitionAsking[Ask, Answered]) askTheNextReplica(
	ctx context.Context,
	putOn PutOn,
) {
	placedAsk, chosen := asking.chooseTheNextAsk()
	if !chosen {
		return
	}
	asking.putTheAsk(ctx, placedAsk, putOn)
}

func (asking *wordPartitionAsking[Ask, Answered]) takeTheCallOutcome(
	ctx context.Context,
	outcome replicaCallOutcome[Ask, Answered],
) {
	asking.endTheCall(outcome)
	asking.recordTheAnswer(outcome)
	asking.askTheNextReplicaWhenNotListing(ctx, outcome)
	asking.countTheListingAnswer(outcome)
	asking.settleWhenNothingIsLeftToAsk()
}

func (asking *wordPartitionAsking[Ask, Answered]) endTheCall(
	outcome replicaCallOutcome[Ask, Answered],
) {
	asking.amountOfCallsOutstanding--
	outcome.call.ended = true
}

func (asking *wordPartitionAsking[Ask, Answered]) recordTheAnswer(
	outcome replicaCallOutcome[Ask, Answered],
) {
	if outcome.answered {
		outcome.call.answer = yacymodel.Some(outcome.answer)
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) askTheNextReplicaWhenNotListing(
	ctx context.Context,
	outcome replicaCallOutcome[Ask, Answered],
) {
	switch {
	case !outcome.answered:
		asking.askTheNextReplica(ctx, PutOnFailure)
	case !outcome.listsDocuments:
		asking.askTheNextReplica(ctx, PutOnEmptyAnswer)
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) countTheListingAnswer(
	outcome replicaCallOutcome[Ask, Answered],
) {
	if !outcome.listsDocuments {
		return
	}
	asking.amountOfListingAnswers++
	if asking.amountOfListingAnswers == asking.partition.amountOfReplicasCoveringAPartition {
		asking.settledBy = SettledByCoverage
		asking.coveringAskPutOn = outcome.call.putOn
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) stopTheHedgeTimers() {
	for _, call := range asking.calls {
		call.hedgeTimer.Stop()
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) askedWordPartition() askedWordPartition[Ask, Answered] {
	return askedWordPartition[Ask, Answered]{
		settledWordPartition: SettledWordPartition{
			SettledBy:               asking.settledBy,
			CoveringAskPutOn:        asking.coveringAskPutOn,
			AmountOfDocumentsListed: asking.amountOfDocumentsListed(),
			AsksPutOn:               asking.asksPutOn(),
		},
		placedAskOutcomes: asking.placedAskOutcomes(),
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) amountOfDocumentsListed() int {
	amountOfDocumentsListed := 0
	for _, call := range asking.calls {
		answer, answered := call.answer.Get()
		if !answered {
			continue
		}
		amountOfDocumentsListed += asking.partition.kind.amountOfDocumentsListedIn(answer)
	}

	return amountOfDocumentsListed
}

func (asking *wordPartitionAsking[Ask, Answered]) asksPutOn() []PutOn {
	asksPutOn := make([]PutOn, 0, len(asking.calls))
	for _, call := range asking.calls {
		asksPutOn = append(asksPutOn, call.putOn)
	}

	return asksPutOn
}

func (asking *wordPartitionAsking[Ask, Answered]) placedAskOutcomes() []placedAskOutcome[Ask, Answered] {
	placedAskOutcomes := make([]placedAskOutcome[Ask, Answered], 0, len(asking.calls))
	for _, call := range asking.calls {
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
