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

func (asking *wordPartitionAsking[Ask, Answered]) settle(
	ctx context.Context,
	firstReplicas []int,
) answeredWordPartition[Ask, Answered] {
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	for _, replica := range firstReplicas {
		asking.putTheReplica(askingContext, replica, PutOnStart)
	}
	asking.settleWhenNoReplicaIsLeft()
	for !asking.settled() {
		asking.takeTheNextEvent(askingContext)
	}
	asking.stopTheHedgeTimers()

	return asking.answeredWordPartition()
}

type wordPartitionAsking[Ask any, Answered any] struct {
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

func (partition wordPartition[Ask, Answered]) asking() *wordPartitionAsking[Ask, Answered] {
	return &wordPartitionAsking[Ask, Answered]{
		partition:    partition,
		hedgesDue:    make(chan int, len(partition.asksInReplicaOrder)),
		callOutcomes: make(chan replicaCallOutcome[Answered], len(partition.asksInReplicaOrder)),
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) reserveTheFirstReplicas() []int {
	firstReplicas := make([]int, 0, asking.partition.replicasCoveringAPartition)
	for range asking.partition.replicasCoveringAPartition {
		replica, reserved := asking.reserveTheNextReplica()
		if !reserved {
			break
		}
		firstReplicas = append(firstReplicas, replica)
	}

	return firstReplicas
}

func (asking *wordPartitionAsking[Ask, Answered]) settled() bool {
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
	if asking.callsEnded[callPlace] {
		return
	}
	asking.askTheNextReplica(ctx, PutOnHedgeDelay)
}

func (asking *wordPartitionAsking[Ask, Answered]) takeTheCallOutcome(
	ctx context.Context,
	outcome replicaCallOutcome[Answered],
) {
	asking.amountOfCallsOutstanding--
	asking.callsEnded[outcome.callPlace] = true
	asking.coverOrAskTheNextReplica(ctx, outcome)
	asking.settleWhenNoReplicaIsLeft()
}

func (asking *wordPartitionAsking[Ask, Answered]) settleWhenNoReplicaIsLeft() {
	if !asking.settled() && asking.amountOfCallsOutstanding == 0 && asking.noReplicaIsLeft() {
		asking.settledBy = SettledByNoReplicaLeft
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) coverOrAskTheNextReplica(
	ctx context.Context,
	outcome replicaCallOutcome[Answered],
) {
	if !outcome.answered {
		asking.askTheNextReplica(ctx, PutOnFailure)

		return
	}
	asking.answerOfEachCall[outcome.callPlace] = yacymodel.Some(outcome.answer)
	if !outcome.covers {
		asking.askTheNextReplica(ctx, PutOnNonCoveringAnswer)

		return
	}
	asking.amountOfCoveringAnswers++
	if asking.amountOfCoveringAnswers == asking.partition.replicasCoveringAPartition {
		asking.settledBy = SettledByCoverage
		asking.coveringAskPutOn = asking.putOnPerCall[outcome.callPlace]
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) askTheNextReplica(
	ctx context.Context,
	putOn PutOn,
) {
	replica, reserved := asking.reserveTheNextReplica()
	if !reserved {
		return
	}
	asking.putTheReplica(ctx, replica, putOn)
}

func (asking *wordPartitionAsking[Ask, Answered]) reserveTheNextReplica() (int, bool) {
	for !asking.noReplicaIsLeft() {
		replica := asking.nextReplica
		asking.nextReplica++
		if asking.partition.askedPeers.reserve(
			asking.partition.kind.peerOf(asking.partition.asksInReplicaOrder[replica]),
		) {
			return replica, true
		}
	}

	return 0, false
}

func (asking *wordPartitionAsking[Ask, Answered]) putTheReplica(
	ctx context.Context,
	replica int,
	putOn PutOn,
) {
	ask := asking.partition.asksInReplicaOrder[replica]
	callPlace := len(asking.putOnPerCall)
	asking.replicaOfEachCall = append(asking.replicaOfEachCall, replica)
	asking.answerOfEachCall = append(asking.answerOfEachCall, yacymodel.None[Answered]())
	asking.putOnPerCall = append(asking.putOnPerCall, putOn)
	asking.callsEnded = append(asking.callsEnded, false)
	asking.amountOfCallsOutstanding++
	asking.hedgeTimers = append(asking.hedgeTimers, time.AfterFunc(
		asking.partition.kind.hedgeDelayOf(ctx, ask),
		func() { asking.hedgesDue <- callPlace },
	))
	go asking.putTheAsk(ctx, callPlace, ask)
}

func (asking *wordPartitionAsking[Ask, Answered]) putTheAsk(
	ctx context.Context,
	callPlace int,
	ask Ask,
) {
	answer, answered := asking.partition.kind.putAsk(ctx, ask)
	asking.callOutcomes <- replicaCallOutcome[Answered]{
		callPlace: callPlace,
		answered:  answered,
		answer:    answer,
		covers:    answered && asking.partition.kind.isCovering(answer),
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) noReplicaIsLeft() bool {
	return asking.nextReplica == len(asking.partition.asksInReplicaOrder)
}

func (asking *wordPartitionAsking[Ask, Answered]) stopTheHedgeTimers() {
	for _, hedgeTimer := range asking.hedgeTimers {
		hedgeTimer.Stop()
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) answeredWordPartition() answeredWordPartition[Ask, Answered] {
	return answeredWordPartition[Ask, Answered]{
		settled: SettledWordPartition{
			SettledBy:               asking.settledBy,
			CoveringAskPutOn:        asking.coveringAskPutOn,
			AmountOfDocumentsListed: asking.amountOfDocumentsListed(),
			AsksPutOn:               asking.putOnPerCall,
		},
		placedAskOutcomes: asking.placedAskOutcomes(),
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) amountOfDocumentsListed() int {
	amountOfDocumentsListed := 0
	for _, answerOfCall := range asking.answerOfEachCall {
		answer, answered := answerOfCall.Get()
		if !answered {
			continue
		}
		amountOfDocumentsListed += asking.partition.kind.amountOfDocumentsListedIn(answer)
	}

	return amountOfDocumentsListed
}

func (asking *wordPartitionAsking[Ask, Answered]) placedAskOutcomes() []placedAskOutcome[Ask, Answered] {
	placedAskOutcomes := make([]placedAskOutcome[Ask, Answered], 0, len(asking.replicaOfEachCall))
	for callPlace, replica := range asking.replicaOfEachCall {
		placedAskOutcomes = append(placedAskOutcomes, placedAskOutcome[Ask, Answered]{
			placeInTheRun: asking.partition.placeOfEachAskInTheRun[replica],
			outcome: peerasks.AskOutcome[Ask, Answered]{
				Ask:    asking.partition.asksInReplicaOrder[replica],
				Put:    true,
				Answer: asking.answerOfEachCall[callPlace],
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
