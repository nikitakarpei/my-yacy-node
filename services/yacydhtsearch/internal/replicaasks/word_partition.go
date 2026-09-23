package replicaasks

import (
	"context"
	"time"
)

type wordPartition[Ask any, Answered any] struct {
	asksInReplicaOrder         []Ask
	kind                       askKind[Ask, Answered]
	replicasCoveringAPartition int
}

type answeredWordPartition[Ask any, Answered any] struct {
	settled SettledWordPartition
	asksPut []Ask
	answers []Answered
}

func (partition wordPartition[Ask, Answered]) settle(
	ctx context.Context,
) answeredWordPartition[Ask, Answered] {
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	asking := partition.asking()
	asking.askTheFirstReplicas(askingContext)
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
	putOnPerReplica          []PutOn
	callsEnded               []bool
	hedgeTimers              []*time.Timer
	amountOfCallsOutstanding int
	amountOfCoveringAnswers  int
	answers                  []Answered
}

func (partition wordPartition[Ask, Answered]) asking() *wordPartitionAsking[Ask, Answered] {
	return &wordPartitionAsking[Ask, Answered]{
		partition:    partition,
		hedgesDue:    make(chan int, len(partition.asksInReplicaOrder)),
		callOutcomes: make(chan replicaCallOutcome[Answered], len(partition.asksInReplicaOrder)),
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) askTheFirstReplicas(ctx context.Context) {
	for range min(
		asking.partition.replicasCoveringAPartition, len(asking.partition.asksInReplicaOrder),
	) {
		asking.askTheNextReplica(ctx, PutOnStart)
	}
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
	case replica := <-asking.hedgesDue:
		asking.takeTheHedgeDue(ctx, replica)
	case outcome := <-asking.callOutcomes:
		asking.takeTheCallOutcome(ctx, outcome)
	case <-ctx.Done():
		asking.settledBy = SettledByDeadline
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) takeTheHedgeDue(
	ctx context.Context,
	replica int,
) {
	if asking.callsEnded[replica] {
		return
	}
	asking.askTheNextReplica(ctx, PutOnHedgeDelay)
}

func (asking *wordPartitionAsking[Ask, Answered]) takeTheCallOutcome(
	ctx context.Context,
	outcome replicaCallOutcome[Answered],
) {
	asking.amountOfCallsOutstanding--
	asking.callsEnded[outcome.replica] = true
	asking.coverOrAskTheNextReplica(ctx, outcome)
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
	asking.answers = append(asking.answers, outcome.answer)
	if !outcome.covers {
		asking.askTheNextReplica(ctx, PutOnNonCoveringAnswer)

		return
	}
	asking.amountOfCoveringAnswers++
	if asking.amountOfCoveringAnswers == asking.partition.replicasCoveringAPartition {
		asking.settledBy = SettledByCoverage
		asking.coveringAskPutOn = asking.putOnPerReplica[outcome.replica]
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) askTheNextReplica(
	ctx context.Context,
	putOn PutOn,
) {
	if asking.noReplicaIsLeft() {
		return
	}
	replica := len(asking.putOnPerReplica)
	ask := asking.partition.asksInReplicaOrder[replica]
	asking.putOnPerReplica = append(asking.putOnPerReplica, putOn)
	asking.callsEnded = append(asking.callsEnded, false)
	asking.amountOfCallsOutstanding++
	asking.hedgeTimers = append(asking.hedgeTimers, time.AfterFunc(
		asking.partition.kind.hedgeDelayOf(ctx, ask),
		func() { asking.hedgesDue <- replica },
	))
	go asking.putTheAsk(ctx, replica, ask)
}

func (asking *wordPartitionAsking[Ask, Answered]) putTheAsk(
	ctx context.Context,
	replica int,
	ask Ask,
) {
	answer, answered := asking.partition.kind.putAsk(ctx, ask)
	asking.callOutcomes <- replicaCallOutcome[Answered]{
		replica:  replica,
		answered: answered,
		answer:   answer,
		covers:   answered && asking.partition.kind.isCovering(answer),
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) noReplicaIsLeft() bool {
	return len(asking.putOnPerReplica) == len(asking.partition.asksInReplicaOrder)
}

func (asking *wordPartitionAsking[Ask, Answered]) stopTheHedgeTimers() {
	for _, hedgeTimer := range asking.hedgeTimers {
		hedgeTimer.Stop()
	}
}

func (asking *wordPartitionAsking[Ask, Answered]) answeredWordPartition() answeredWordPartition[Ask, Answered] {
	amountOfDocumentsListed := 0
	for _, answer := range asking.answers {
		amountOfDocumentsListed += asking.partition.kind.amountOfDocumentsListedIn(answer)
	}

	return answeredWordPartition[Ask, Answered]{
		settled: SettledWordPartition{
			SettledBy:               asking.settledBy,
			CoveringAskPutOn:        asking.coveringAskPutOn,
			AmountOfDocumentsListed: amountOfDocumentsListed,
			AsksPutOn:               asking.putOnPerReplica,
		},
		asksPut: asking.partition.asksInReplicaOrder[:len(asking.putOnPerReplica)],
		answers: asking.answers,
	}
}

type replicaCallOutcome[Answered any] struct {
	replica  int
	answered bool
	covers   bool
	answer   Answered
}
