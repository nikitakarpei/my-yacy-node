package replicaasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartition[Ask any, Answered any] struct {
	asksInReplicaOrder         []Ask
	placeOfEachAskInTheRun     []int
	kind                       askKind[Ask, Answered]
	replicasCoveringAPartition int
	askedPeers                 *askedPeers
}

type answeredWordPartition[Ask any, Answered any] struct {
	settled           SettledWordPartition
	placedAskOutcomes []placedAskOutcome[Ask, Answered]
}

type placedAskOutcome[Ask any, Answered any] struct {
	placeInTheRun int
	outcome       peerasks.AskOutcome[Ask, Answered]
}

func (unsettled *unsettledWordPartition[Ask, Answered]) settle(
	ctx context.Context,
	firstReplicas []int,
) answeredWordPartition[Ask, Answered] {
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	for _, replica := range firstReplicas {
		unsettled.putTheReplica(askingContext, replica, PutOnStart)
	}
	unsettled.settleWhenNoReplicaIsLeft()
	for !unsettled.settled() {
		unsettled.takeTheNextEvent(askingContext)
	}
	unsettled.stopTheHedgeTimers()

	return unsettled.answeredWordPartition()
}

type unsettledWordPartition[Ask any, Answered any] struct {
	partition                wordPartition[Ask, Answered]
	hedgesDue                chan int
	callOutcomes             chan replicaCallOutcome[Answered]
	settledBy                SettledBy
	coveringAskPutOn         PutOn
	putOnPerCall             []PutOn
	callsEnded               []bool
	hedgeTimers              []*time.Timer
	amountOfCallsOutstanding int
	amountOfCoveringAnswers  int
	nextReplica              int
	replicaOfEachCall        []int
	answerOfEachCall         []yacymodel.Optional[Answered]
}

func (partition wordPartition[Ask, Answered]) unsettled() *unsettledWordPartition[Ask, Answered] {
	return &unsettledWordPartition[Ask, Answered]{
		partition:    partition,
		hedgesDue:    make(chan int, len(partition.asksInReplicaOrder)),
		callOutcomes: make(chan replicaCallOutcome[Answered], len(partition.asksInReplicaOrder)),
	}
}

func (unsettled *unsettledWordPartition[Ask, Answered]) reserveTheFirstReplicas() []int {
	firstReplicas := make([]int, 0, unsettled.partition.replicasCoveringAPartition)
	for range unsettled.partition.replicasCoveringAPartition {
		replica, reserved := unsettled.reserveTheNextReplica()
		if !reserved {
			break
		}
		firstReplicas = append(firstReplicas, replica)
	}

	return firstReplicas
}

func (unsettled *unsettledWordPartition[Ask, Answered]) settled() bool {
	return unsettled.settledBy != ""
}

func (unsettled *unsettledWordPartition[Ask, Answered]) takeTheNextEvent(ctx context.Context) {
	if ctx.Err() != nil {
		unsettled.settledBy = SettledByDeadline

		return
	}
	select {
	case callPlace := <-unsettled.hedgesDue:
		unsettled.takeTheHedgeDue(ctx, callPlace)
	case outcome := <-unsettled.callOutcomes:
		unsettled.takeTheCallOutcome(ctx, outcome)
	case <-ctx.Done():
		unsettled.settledBy = SettledByDeadline
	}
}

func (unsettled *unsettledWordPartition[Ask, Answered]) takeTheHedgeDue(
	ctx context.Context,
	callPlace int,
) {
	if unsettled.callsEnded[callPlace] {
		return
	}
	unsettled.askTheNextReplica(ctx, PutOnHedgeDelay)
}

func (unsettled *unsettledWordPartition[Ask, Answered]) takeTheCallOutcome(
	ctx context.Context,
	outcome replicaCallOutcome[Answered],
) {
	unsettled.amountOfCallsOutstanding--
	unsettled.callsEnded[outcome.callPlace] = true
	unsettled.coverOrAskTheNextReplica(ctx, outcome)
	unsettled.settleWhenNoReplicaIsLeft()
}

func (unsettled *unsettledWordPartition[Ask, Answered]) settleWhenNoReplicaIsLeft() {
	if !unsettled.settled() && unsettled.amountOfCallsOutstanding == 0 && unsettled.noReplicaIsLeft() {
		unsettled.settledBy = SettledByNoReplicaLeft
	}
}

func (unsettled *unsettledWordPartition[Ask, Answered]) coverOrAskTheNextReplica(
	ctx context.Context,
	outcome replicaCallOutcome[Answered],
) {
	if !outcome.answered {
		unsettled.askTheNextReplica(ctx, PutOnFailure)

		return
	}
	unsettled.answerOfEachCall[outcome.callPlace] = yacymodel.Some(outcome.answer)
	if !outcome.covers {
		unsettled.askTheNextReplica(ctx, PutOnNonCoveringAnswer)

		return
	}
	unsettled.amountOfCoveringAnswers++
	if unsettled.amountOfCoveringAnswers == unsettled.partition.replicasCoveringAPartition {
		unsettled.settledBy = SettledByCoverage
		unsettled.coveringAskPutOn = unsettled.putOnPerCall[outcome.callPlace]
	}
}

func (unsettled *unsettledWordPartition[Ask, Answered]) askTheNextReplica(
	ctx context.Context,
	putOn PutOn,
) {
	replica, reserved := unsettled.reserveTheNextReplica()
	if !reserved {
		return
	}
	unsettled.putTheReplica(ctx, replica, putOn)
}

func (unsettled *unsettledWordPartition[Ask, Answered]) reserveTheNextReplica() (int, bool) {
	for !unsettled.noReplicaIsLeft() {
		replica := unsettled.nextReplica
		unsettled.nextReplica++
		if unsettled.partition.askedPeers.reserve(
			unsettled.partition.kind.peerOf(unsettled.partition.asksInReplicaOrder[replica]),
		) {
			return replica, true
		}
	}

	return 0, false
}

func (unsettled *unsettledWordPartition[Ask, Answered]) putTheReplica(
	ctx context.Context,
	replica int,
	putOn PutOn,
) {
	ask := unsettled.partition.asksInReplicaOrder[replica]
	callPlace := len(unsettled.putOnPerCall)
	unsettled.replicaOfEachCall = append(unsettled.replicaOfEachCall, replica)
	unsettled.answerOfEachCall = append(unsettled.answerOfEachCall, yacymodel.None[Answered]())
	unsettled.putOnPerCall = append(unsettled.putOnPerCall, putOn)
	unsettled.callsEnded = append(unsettled.callsEnded, false)
	unsettled.amountOfCallsOutstanding++
	unsettled.hedgeTimers = append(unsettled.hedgeTimers, time.AfterFunc(
		unsettled.partition.kind.hedgeDelayOf(ctx, ask),
		func() { unsettled.hedgesDue <- callPlace },
	))
	go unsettled.putTheAsk(ctx, callPlace, ask)
}

func (unsettled *unsettledWordPartition[Ask, Answered]) putTheAsk(
	ctx context.Context,
	callPlace int,
	ask Ask,
) {
	answer, answered := unsettled.partition.kind.putAsk(ctx, ask)
	unsettled.callOutcomes <- replicaCallOutcome[Answered]{
		callPlace: callPlace,
		answered:  answered,
		answer:    answer,
		covers:    answered && unsettled.partition.kind.isCovering(answer),
	}
}

func (unsettled *unsettledWordPartition[Ask, Answered]) noReplicaIsLeft() bool {
	return unsettled.nextReplica == len(unsettled.partition.asksInReplicaOrder)
}

func (unsettled *unsettledWordPartition[Ask, Answered]) stopTheHedgeTimers() {
	for _, hedgeTimer := range unsettled.hedgeTimers {
		hedgeTimer.Stop()
	}
}

func (unsettled *unsettledWordPartition[Ask, Answered]) answeredWordPartition() answeredWordPartition[Ask, Answered] {
	return answeredWordPartition[Ask, Answered]{
		settled: SettledWordPartition{
			SettledBy:               unsettled.settledBy,
			CoveringAskPutOn:        unsettled.coveringAskPutOn,
			AmountOfDocumentsListed: unsettled.amountOfDocumentsListed(),
			AsksPutOn:               unsettled.putOnPerCall,
		},
		placedAskOutcomes: unsettled.placedAskOutcomes(),
	}
}

func (unsettled *unsettledWordPartition[Ask, Answered]) amountOfDocumentsListed() int {
	amountOfDocumentsListed := 0
	for _, answerOfCall := range unsettled.answerOfEachCall {
		answer, answered := answerOfCall.Get()
		if !answered {
			continue
		}
		amountOfDocumentsListed += unsettled.partition.kind.amountOfDocumentsListedIn(answer)
	}

	return amountOfDocumentsListed
}

func (unsettled *unsettledWordPartition[Ask, Answered]) placedAskOutcomes() []placedAskOutcome[Ask, Answered] {
	placedAskOutcomes := make([]placedAskOutcome[Ask, Answered], 0, len(unsettled.replicaOfEachCall))
	for callPlace, replica := range unsettled.replicaOfEachCall {
		placedAskOutcomes = append(placedAskOutcomes, placedAskOutcome[Ask, Answered]{
			placeInTheRun: unsettled.partition.placeOfEachAskInTheRun[replica],
			outcome: peerasks.AskOutcome[Ask, Answered]{
				Ask:    unsettled.partition.asksInReplicaOrder[replica],
				Put:    true,
				Answer: unsettled.answerOfEachCall[callPlace],
			},
		})
	}

	return placedAskOutcomes
}

type replicaCallOutcome[Answered any] struct {
	callPlace int
	answered  bool
	covers    bool
	answer    Answered
}
