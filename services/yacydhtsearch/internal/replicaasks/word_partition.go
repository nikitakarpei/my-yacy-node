package replicaasks

import (
	"context"
	"time"
)

type wordPartition[Ask any, Answered any] struct {
	replicas                   []Ask
	calls                      replicaCalls[Ask, Answered]
	replicasCoveringAPartition int
}

type settledWordPartition[Answered any] struct {
	settledBy               SettledBy
	amountOfDocumentsListed int
	putAs                   []PutAs
	answers                 []Answered
}

func (partition wordPartition[Ask, Answered]) settle(
	ctx context.Context,
) settledWordPartition[Answered] {
	callingContext, stopCalling := context.WithCancel(ctx)
	defer stopCalling()

	asked := partition.askedReplicas()
	asked.putTheFirstReplicas(callingContext)
	for asked.settledBy == "" {
		asked.takeTheNextEvent(callingContext)
	}
	asked.stopTheHedgeTimers()

	return asked.settledWordPartition()
}

type askedReplicas[Ask any, Answered any] struct {
	partition            wordPartition[Ask, Answered]
	events               chan replicaCallEvent[Answered]
	settledBy            SettledBy
	putAs                []PutAs
	replicasThatCameBack []bool
	hedgeTimers          []*time.Timer
	callsOutstanding     int
	listingsCounted      int
	answers              []Answered
}

func (partition wordPartition[Ask, Answered]) askedReplicas() *askedReplicas[Ask, Answered] {
	return &askedReplicas[Ask, Answered]{
		partition: partition,
		events:    make(chan replicaCallEvent[Answered], 2*len(partition.replicas)),
	}
}

func (asked *askedReplicas[Ask, Answered]) putTheFirstReplicas(ctx context.Context) {
	for range min(asked.partition.replicasCoveringAPartition, len(asked.partition.replicas)) {
		asked.putTheNextReplica(ctx, PutAsFirst)
	}
}

func (asked *askedReplicas[Ask, Answered]) takeTheNextEvent(ctx context.Context) {
	if ctx.Err() != nil {
		asked.settledBy = SettledByDeadline

		return
	}
	select {
	case event := <-asked.events:
		asked.takeTheEvent(ctx, event)
	case <-ctx.Done():
		asked.settledBy = SettledByDeadline
	}
}

func (asked *askedReplicas[Ask, Answered]) takeTheEvent(
	ctx context.Context,
	event replicaCallEvent[Answered],
) {
	if event.hedgeIsDue {
		asked.takeTheHedgeDue(ctx, event.replica)

		return
	}
	asked.callsOutstanding--
	asked.replicasThatCameBack[event.replica] = true
	asked.takeTheCallOutcome(ctx, event)
	if asked.settledBy == "" && asked.callsOutstanding == 0 && asked.noReplicaIsLeft() {
		asked.settledBy = SettledByExhausted
	}
}

func (asked *askedReplicas[Ask, Answered]) takeTheHedgeDue(ctx context.Context, replica int) {
	if asked.replicasThatCameBack[replica] {
		return
	}
	asked.putTheNextReplica(ctx, PutAsHedge)
}

func (asked *askedReplicas[Ask, Answered]) takeTheCallOutcome(
	ctx context.Context,
	event replicaCallEvent[Answered],
) {
	if !event.answered {
		asked.putTheNextReplica(ctx, PutAsAfterAFailure)

		return
	}
	asked.answers = append(asked.answers, event.answer)
	if !event.listsDocuments {
		asked.putTheNextReplica(ctx, PutAsAfterAnEmptyAnswer)

		return
	}
	asked.listingsCounted++
	if asked.listingsCounted == asked.partition.replicasCoveringAPartition {
		asked.settledBy = settledByOf(asked.putAs[event.replica])
	}
}

func (asked *askedReplicas[Ask, Answered]) putTheNextReplica(ctx context.Context, putAs PutAs) {
	if asked.noReplicaIsLeft() {
		return
	}
	replica := len(asked.putAs)
	ask := asked.partition.replicas[replica]
	asked.putAs = append(asked.putAs, putAs)
	asked.replicasThatCameBack = append(asked.replicasThatCameBack, false)
	asked.callsOutstanding++
	asked.hedgeTimers = append(asked.hedgeTimers, time.AfterFunc(
		asked.partition.calls.hedgeDelayOf(ctx, ask),
		func() { asked.events <- replicaCallEvent[Answered]{replica: replica, hedgeIsDue: true} },
	))
	go asked.putTheAsk(ctx, replica, ask)
}

func (asked *askedReplicas[Ask, Answered]) putTheAsk(
	ctx context.Context,
	replica int,
	ask Ask,
) {
	answer, answered := asked.partition.calls.putAsk(ctx, ask)
	asked.events <- replicaCallEvent[Answered]{
		replica:        replica,
		answered:       answered,
		answer:         answer,
		listsDocuments: answered && asked.partition.calls.amountOfDocumentsListedIn(answer) > 0,
	}
}

func (asked *askedReplicas[Ask, Answered]) noReplicaIsLeft() bool {
	return len(asked.putAs) == len(asked.partition.replicas)
}

func (asked *askedReplicas[Ask, Answered]) stopTheHedgeTimers() {
	for _, hedgeTimer := range asked.hedgeTimers {
		hedgeTimer.Stop()
	}
}

func (asked *askedReplicas[Ask, Answered]) settledWordPartition() settledWordPartition[Answered] {
	amountOfDocumentsListed := 0
	for _, answer := range asked.answers {
		amountOfDocumentsListed += asked.partition.calls.amountOfDocumentsListedIn(answer)
	}

	return settledWordPartition[Answered]{
		settledBy:               asked.settledBy,
		amountOfDocumentsListed: amountOfDocumentsListed,
		putAs:                   asked.putAs,
		answers:                 asked.answers,
	}
}

type replicaCallEvent[Answered any] struct {
	replica        int
	hedgeIsDue     bool
	answered       bool
	listsDocuments bool
	answer         Answered
}
