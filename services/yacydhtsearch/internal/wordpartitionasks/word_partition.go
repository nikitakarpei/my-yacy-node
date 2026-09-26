package wordpartitionasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type placedAsk struct {
	ask                 peerasks.SearchDocumentsAsk
	placeInReplicaOrder int
}

type wordPartition struct {
	replicaAsks              Asks
	chosenPeers              *chosenPeers
	asksInReplicaOrder       []placedAsk
	asksLeft                 []placedAsk
	firstAsks                []placedAsk
	calls                    []*replicaCall
	hedgesDue                chan *replicaCall
	callOutcomes             chan replicaCallOutcome
	settledBy                SettledBy
	coveringAskPutOn         PutOn
	amountOfCallsOutstanding int
	amountOfCoveringAnswers  int
}

type replicaCall struct {
	placedAsk  placedAsk
	putOn      PutOn
	ended      bool
	hedgeTimer *time.Timer
	answer     yacymodel.Optional[peerasks.AnsweredSearchDocumentsAsk]
}

type replicaCallOutcome struct {
	call               *replicaCall
	answered           bool
	coversThePartition bool
	answer             peerasks.AnsweredSearchDocumentsAsk
}

func wordPartitionOf(
	asksInReplicaOrder []placedAsk,
	replicaAsks Asks,
	chosenPeers *chosenPeers,
) *wordPartition {
	return &wordPartition{
		replicaAsks:        replicaAsks,
		chosenPeers:        chosenPeers,
		asksInReplicaOrder: asksInReplicaOrder,
		asksLeft:           asksInReplicaOrder,
		hedgesDue:          make(chan *replicaCall, len(asksInReplicaOrder)),
		callOutcomes:       make(chan replicaCallOutcome, len(asksInReplicaOrder)),
	}
}

func (partition *wordPartition) chooseTheFirstAsks() {
	for range partition.replicaAsks.amountOfReplicasCoveringAPartition {
		placedAsk, chosen := partition.chooseTheNextAsk()
		if !chosen {
			return
		}
		partition.firstAsks = append(partition.firstAsks, placedAsk)
	}
}

func (partition *wordPartition) chooseTheNextAsk() (placedAsk, bool) {
	for !partition.noAskIsLeft() {
		placedAsk := partition.asksLeft[0]
		partition.asksLeft = partition.asksLeft[1:]
		if partition.chosenPeers.choose(placedAsk.ask.Peer.Hash) {
			return placedAsk, true
		}
	}

	return placedAsk{}, false
}

func (partition *wordPartition) noAskIsLeft() bool {
	return len(partition.asksLeft) == 0
}

func (partition *wordPartition) askUntilSettled(ctx context.Context) {
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

func (partition *wordPartition) putTheAsk(
	ctx context.Context,
	placedAsk placedAsk,
	putOn PutOn,
) {
	call := &replicaCall{
		placedAsk: placedAsk,
		putOn:     putOn,
		answer:    yacymodel.None[peerasks.AnsweredSearchDocumentsAsk](),
	}
	call.hedgeTimer = time.AfterFunc(
		partition.replicaAsks.hedgeDelay.HedgeDelayOf(ctx, placedAsk.ask.Peer),
		func() { partition.hedgesDue <- call },
	)
	partition.calls = append(partition.calls, call)
	partition.amountOfCallsOutstanding++
	go partition.callTheReplica(ctx, call, placedAsk.ask)
}

func (partition *wordPartition) callTheReplica(
	ctx context.Context,
	call *replicaCall,
	ask peerasks.SearchDocumentsAsk,
) {
	answeredAsks := partition.replicaAsks.peerCalls.AskForSearchDocuments(
		ctx,
		[]peerasks.SearchDocumentsAsk{ask},
	)
	if len(answeredAsks) == 0 {
		partition.callOutcomes <- replicaCallOutcome{call: call}

		return
	}
	partition.callOutcomes <- replicaCallOutcome{
		call:               call,
		answered:           true,
		answer:             answeredAsks[0],
		coversThePartition: coversThePartition(answeredAsks[0]),
	}
}

func coversThePartition(answeredAsk peerasks.AnsweredSearchDocumentsAsk) bool {
	return amountOfDocumentsListedIn(answeredAsk) > 0 || answeredAsk.PeerSearched
}

func amountOfDocumentsListedIn(answeredAsk peerasks.AnsweredSearchDocumentsAsk) int {
	if len(answeredAsk.Abstract) > 0 {
		return len(answeredAsk.Abstract)
	}
	if len(answeredAsk.MatchedDocuments) > 0 {
		return len(answeredAsk.MatchedDocuments)
	}
	amountHeld, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
	if !counted {
		return 0
	}

	return max(0, amountHeld)
}

func (partition *wordPartition) settleWhenNothingIsLeftToAsk() {
	if !partition.isSettled() && partition.amountOfCallsOutstanding == 0 &&
		partition.noAskIsLeft() {
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
	placedAsk, chosen := partition.chooseTheNextAsk()
	if !chosen {
		return
	}
	partition.putTheAsk(ctx, placedAsk, putOn)
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
		call.hedgeTimer.Stop()
	}
}

func (partition *wordPartition) settledWordPartition() SettledWordPartition {
	callOfEachPlace := make(map[int]*replicaCall, len(partition.calls))
	for _, call := range partition.calls {
		callOfEachPlace[call.placedAsk.placeInReplicaOrder] = call
	}
	askOutcomes := make(peerasks.SearchDocumentsAskOutcomes, 0, len(partition.asksInReplicaOrder))
	for _, placedAsk := range partition.asksInReplicaOrder {
		askOutcome := peerasks.SearchDocumentsAskOutcome{Ask: placedAsk.ask}
		if call, put := callOfEachPlace[placedAsk.placeInReplicaOrder]; put {
			askOutcome.Put = true
			askOutcome.Answer = call.answer
		}
		askOutcomes = append(askOutcomes, askOutcome)
	}

	return SettledWordPartition{AskOutcomes: askOutcomes}
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
