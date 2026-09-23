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
	reservedPeers                      *reservedPeers
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
	calls                    []replicaCall[Ask, Answered]
	hedgesDue                chan int
	callOutcomes             chan replicaCallOutcome[Answered]
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

type replicaCallOutcome[Answered any] struct {
	callPlace      int
	answered       bool
	listsDocuments bool
	answer         Answered
}

func (partition wordPartition[Ask, Answered]) asking() *wordPartitionAsking[Ask, Answered] {
	return &wordPartitionAsking[Ask, Answered]{
		partition:    partition,
		asksLeft:     partition.asksInReplicaOrder,
		hedgesDue:    make(chan int, len(partition.asksInReplicaOrder)),
		callOutcomes: make(chan replicaCallOutcome[Answered], len(partition.asksInReplicaOrder)),
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) reserveTheFirstAsks() {
	for range asking.partition.amountOfReplicasCoveringAPartition {
		placedAsk, reserved := asking.reserveTheNextAsk()
		if !reserved {
			return
		}
		asking.firstAsks = append(asking.firstAsks, placedAsk)
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) reserveTheNextAsk() (placedAsk[Ask], bool) {
	for !asking.noAskIsLeft() {
		placedAsk := asking.asksLeft[0]
		asking.asksLeft = asking.asksLeft[1:]
		if asking.partition.reservedPeers.reserve(asking.partition.peerOf(placedAsk)) {
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

func (asking *wordPartitionAsking[Ask, Answered]) settle(
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
	callPlace := len(asking.calls)
	asking.calls = append(asking.calls, replicaCall[Ask, Answered]{
		placedAsk: placedAsk,
		putOn:     putOn,
		answer:    yacymodel.None[Answered](),
		hedgeTimer: time.AfterFunc(
			asking.partition.kind.hedgeDelayOf(ctx, placedAsk.ask),
			func() { asking.hedgesDue <- callPlace },
		),
	})
	asking.amountOfCallsOutstanding++
	go asking.callTheReplica(ctx, callPlace, placedAsk.ask)
}

func (asking *wordPartitionAsking[Ask, Answered]) callTheReplica(
	ctx context.Context,
	callPlace int,
	ask Ask,
) {
	answer, answered := asking.partition.kind.putAsk(ctx, ask)
	asking.callOutcomes <- replicaCallOutcome[Answered]{
		callPlace:      callPlace,
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
	case callPlace := <-asking.hedgesDue:
		asking.takeTheHedgeDue(ctx, callPlace)
	case outcome := <-asking.callOutcomes:
		asking.takeTheCallOutcome(ctx, outcome)
	case <-ctx.Done():
		asking.settledBy = SettledByDeadline
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) takeTheHedgeDue(
	ctx context.Context,
	callPlace int,
) {
	if asking.calls[callPlace].ended {
		return
	}
	asking.askTheNextReplica(ctx, PutOnHedgeDelay)
}

func (asking *wordPartitionAsking[Ask, Answered]) askTheNextReplica(
	ctx context.Context,
	putOn PutOn,
) {
	placedAsk, reserved := asking.reserveTheNextAsk()
	if !reserved {
		return
	}
	asking.putTheAsk(ctx, placedAsk, putOn)
}

func (asking *wordPartitionAsking[Ask, Answered]) takeTheCallOutcome(
	ctx context.Context,
	outcome replicaCallOutcome[Answered],
) {
	asking.endTheCall(outcome)
	asking.recordTheAnswer(outcome)
	asking.askTheNextReplicaWhenNotListing(ctx, outcome)
	asking.countTheListingAnswer(outcome)
	asking.settleWhenNothingIsLeftToAsk()
}

func (asking *wordPartitionAsking[Ask, Answered]) endTheCall(outcome replicaCallOutcome[Answered]) {
	asking.amountOfCallsOutstanding--
	asking.calls[outcome.callPlace].ended = true
}

func (asking *wordPartitionAsking[Ask, Answered]) recordTheAnswer(
	outcome replicaCallOutcome[Answered],
) {
	if outcome.answered {
		asking.calls[outcome.callPlace].answer = yacymodel.Some(outcome.answer)
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) askTheNextReplicaWhenNotListing(
	ctx context.Context,
	outcome replicaCallOutcome[Answered],
) {
	switch {
	case !outcome.answered:
		asking.askTheNextReplica(ctx, PutOnFailure)
	case !outcome.listsDocuments:
		asking.askTheNextReplica(ctx, PutOnEmptyAnswer)
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) countTheListingAnswer(
	outcome replicaCallOutcome[Answered],
) {
	if !outcome.listsDocuments {
		return
	}
	asking.amountOfListingAnswers++
	if asking.amountOfListingAnswers == asking.partition.amountOfReplicasCoveringAPartition {
		asking.settledBy = SettledByCoverage
		asking.coveringAskPutOn = asking.calls[outcome.callPlace].putOn
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
