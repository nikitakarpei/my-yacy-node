package wordpartitionasks

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartition struct {
	replicaAsks              Asks
	chosenPeers              *chosenPeers
	ask                      Ask
	replicasLeft             []peerdirectory.AskablePeer
	firstReplicas            []peerdirectory.AskablePeer
	calls                    []*replicaCall
	hedgesDue                chan *replicaCall
	callOutcomes             chan replicaCallOutcome
	settledBy                SettledBy
	coveringAskPutOn         PutOn
	amountOfCallsOutstanding int
	amountOfCoveringAnswers  int
}

type replicaCall struct {
	replica        peerdirectory.AskablePeer
	putOn          PutOn
	ended          bool
	stopHedgeTimer func()
	answer         yacymodel.Optional[ReplicaAnswer]
}

type replicaCallOutcome struct {
	call               *replicaCall
	answered           bool
	coversThePartition bool
	answer             ReplicaAnswer
}

func wordPartitionOf(ask Ask, replicaAsks Asks, chosenPeers *chosenPeers) *wordPartition {
	return &wordPartition{
		replicaAsks:  replicaAsks,
		chosenPeers:  chosenPeers,
		ask:          ask,
		replicasLeft: ask.ReplicasInOrder,
		hedgesDue:    make(chan *replicaCall, len(ask.ReplicasInOrder)),
		callOutcomes: make(chan replicaCallOutcome, len(ask.ReplicasInOrder)),
	}
}

func (partition *wordPartition) chooseTheFirstReplicas() {
	for range partition.replicaAsks.amountOfReplicasCoveringAPartition {
		replica, chosen := partition.chooseTheNextReplica()
		if !chosen {
			return
		}
		partition.firstReplicas = append(partition.firstReplicas, replica)
	}
}

func (partition *wordPartition) chooseTheNextReplica() (peerdirectory.AskablePeer, bool) {
	for !partition.noReplicaIsLeft() {
		replica := partition.replicasLeft[0]
		partition.replicasLeft = partition.replicasLeft[1:]
		if partition.chosenPeers.choose(replica.Hash) {
			return replica, true
		}
	}

	return peerdirectory.AskablePeer{}, false
}

func (partition *wordPartition) noReplicaIsLeft() bool {
	return len(partition.replicasLeft) == 0
}

func (partition *wordPartition) askUntilSettled(ctx context.Context) {
	askingContext, stopAsking := context.WithCancel(ctx)
	defer stopAsking()

	for _, replica := range partition.firstReplicas {
		partition.putTheAsk(askingContext, replica, PutOnStart)
	}
	partition.settleWhenNothingIsLeftToAsk()
	for !partition.isSettled() {
		partition.takeTheNextEvent(askingContext)
	}
	partition.stopTheHedgeTimers()
}

func (partition *wordPartition) putTheAsk(
	ctx context.Context,
	replica peerdirectory.AskablePeer,
	putOn PutOn,
) {
	call := &replicaCall{
		replica: replica,
		putOn:   putOn,
		answer:  yacymodel.None[ReplicaAnswer](),
	}
	call.stopHedgeTimer = partition.replicaAsks.startTheHedgeTimer(
		ctx,
		replica,
		func() { partition.hedgesDue <- call },
	)
	partition.calls = append(partition.calls, call)
	partition.amountOfCallsOutstanding++
	go partition.callTheReplica(ctx, call)
}

func (partition *wordPartition) callTheReplica(ctx context.Context, call *replicaCall) {
	answer, answered := partition.replicaAsks.askTheReplica(ctx, partition.ask, call.replica)
	if !answered {
		partition.callOutcomes <- replicaCallOutcome{call: call}

		return
	}
	partition.callOutcomes <- replicaCallOutcome{
		call:               call,
		answered:           true,
		answer:             answer,
		coversThePartition: coversThePartition(answer),
	}
}

func coversThePartition(answer ReplicaAnswer) bool {
	return amountOfDocumentsListedIn(answer) > 0 || answer.Searched
}

func amountOfDocumentsListedIn(answer ReplicaAnswer) int {
	if len(answer.ListedDocuments) > 0 {
		return len(answer.ListedDocuments)
	}
	amountHeld, counted := answer.AmountOfDocumentsHeld.Get()
	if !counted {
		return 0
	}

	return max(0, amountHeld)
}

func (partition *wordPartition) settleWhenNothingIsLeftToAsk() {
	if !partition.isSettled() && partition.amountOfCallsOutstanding == 0 &&
		partition.noReplicaIsLeft() {
		partition.settledBy = SettledByNoReplicaLeft
	}
}

func (partition *wordPartition) isSettled() bool {
	return partition.settledBy != ""
}

func (partition *wordPartition) takeTheNextEvent(ctx context.Context) {
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

func (partition *wordPartition) takeTheHedgeDue(ctx context.Context, call *replicaCall) {
	if call.ended {
		return
	}
	partition.askTheNextReplica(ctx, PutOnHedgeDelay)
}

func (partition *wordPartition) askTheNextReplica(ctx context.Context, putOn PutOn) {
	replica, chosen := partition.chooseTheNextReplica()
	if !chosen {
		return
	}
	partition.putTheAsk(ctx, replica, putOn)
}

func (partition *wordPartition) takeTheCallOutcome(
	ctx context.Context,
	outcome replicaCallOutcome,
) {
	partition.endTheCall(outcome)
	partition.recordTheAnswer(outcome)
	partition.askTheNextReplicaWhenNotCovering(ctx, outcome)
	partition.countTheCoveringAnswer(outcome)
	partition.settleWhenNothingIsLeftToAsk()
}

func (partition *wordPartition) endTheCall(outcome replicaCallOutcome) {
	partition.amountOfCallsOutstanding--
	outcome.call.ended = true
}

func (partition *wordPartition) recordTheAnswer(outcome replicaCallOutcome) {
	if outcome.answered {
		outcome.call.answer = yacymodel.Some(outcome.answer)
	}
}

func (partition *wordPartition) askTheNextReplicaWhenNotCovering(
	ctx context.Context,
	outcome replicaCallOutcome,
) {
	switch {
	case !outcome.answered:
		partition.askTheNextReplica(ctx, PutOnFailure)
	case !outcome.coversThePartition:
		partition.askTheNextReplica(ctx, PutOnEmptyAnswer)
	}
}

func (partition *wordPartition) countTheCoveringAnswer(outcome replicaCallOutcome) {
	if !outcome.coversThePartition {
		return
	}
	partition.amountOfCoveringAnswers++
	if partition.amountOfCoveringAnswers ==
		partition.replicaAsks.amountOfReplicasCoveringAPartition {
		partition.settledBy = SettledByCoverage
		partition.coveringAskPutOn = outcome.call.putOn
	}
}

func (partition *wordPartition) stopTheHedgeTimers() {
	for _, call := range partition.calls {
		call.stopHedgeTimer()
	}
}

func (partition *wordPartition) settledAsk() SettledAsk {
	answers := make([]ReplicaAnswer, 0, len(partition.calls))
	for _, call := range partition.calls {
		answer, answered := call.answer.Get()
		if !answered {
			continue
		}
		answers = append(answers, answer)
	}

	return SettledAsk{Ask: partition.ask, Answers: answers}
}

func (partition *wordPartition) performedWordPartition() PerformedWordPartition {
	return PerformedWordPartition{
		SettledBy:               partition.settledBy,
		CoveringAskPutOn:        partition.coveringAskPutOn,
		AmountOfDocumentsListed: partition.amountOfDocumentsListed(),
		AsksPutOn:               partition.asksPutOn(),
	}
}

func (partition *wordPartition) amountOfDocumentsListed() int {
	amountOfDocumentsListed := 0
	for _, call := range partition.calls {
		answer, answered := call.answer.Get()
		if !answered {
			continue
		}
		amountOfDocumentsListed += amountOfDocumentsListedIn(answer)
	}

	return amountOfDocumentsListed
}

func (partition *wordPartition) asksPutOn() []PutOn {
	asksPutOn := make([]PutOn, 0, len(partition.calls))
	for _, call := range partition.calls {
		asksPutOn = append(asksPutOn, call.putOn)
	}

	return asksPutOn
}
