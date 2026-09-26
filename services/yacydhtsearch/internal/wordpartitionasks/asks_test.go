package wordpartitionasks_test

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	hedgeDelayOfTheTests    = 40 * time.Millisecond
	amountOfTimersBuffered  = 16
	amountOfCancelsBuffered = 16
)

func TestTheFirstReplicasOfEveryWordPartitionAreAskedAndNoMore(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 3}, "berlin-two": {documentsListed: 2},
		"berlin-three": {documentsListed: 1}, "weather-one": {documentsListed: 4},
		"weather-two": {documentsListed: 5}, "weather-three": {documentsListed: 6},
	}, 2)

	answers := asking.answersOf(t.Context(), append(
		asksForTheWord("berlin", 1, "berlin-one", "berlin-two", "berlin-three"),
		asksForTheWord("weather", 2, "weather-one", "weather-two", "weather-three")...,
	))

	if len(answers) != 4 {
		t.Fatalf("the run answered %d asks, want four", len(answers))
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two", "weather-one", "weather-two")
	asking.observer.wantSettledBy(
		t,
		wordpartitionasks.SettledByCoverage,
		wordpartitionasks.SettledByCoverage,
	)
	asking.observer.wantCoveringAskPutOn(
		t,
		wordpartitionasks.PutOnStart,
		wordpartitionasks.PutOnStart,
	)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByCoverage)
}

func TestOneReplicaCoveringAPartitionLeavesTheOtherReplicasUnasked(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 3}, "berlin-two": {documentsListed: 2},
	}, 1)

	answers := asking.answersOf(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 1 || answers[0].Replica.Address != "berlin-one" {
		t.Fatalf("the run = %+v, want the first replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one")
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByCoverage)
	asking.observer.wantCoveringAskPutOn(t, wordpartitionasks.PutOnStart)
	asking.observer.wantAmountOfDocumentsListed(t, 3)
}

func TestAnEmptyAnswerWithoutASearchAsksTheNextReplicaAtOnce(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 0}, "berlin-two": {documentsListed: 2},
	}, 1)

	answers := asking.answersOf(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 2 {
		t.Fatalf("the run answered %d asks, want both replicas", len(answers))
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.observer.wantCoveringAskPutOn(t, wordpartitionasks.PutOnEmptyAnswer)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart, wordpartitionasks.PutOnEmptyAnswer)
}

func TestAnEmptyAnswerOfAPeerThatSearchedSettlesTheWordPartition(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"bananaphotos-one": {searched: true}, "bananaphotos-two": {documentsListed: 2},
	}, 1)

	answers := asking.answersOf(
		t.Context(), asksForTheWord("bananaphotos", 1, "bananaphotos-one", "bananaphotos-two"),
	)

	if len(answers) != 1 || answers[0].Replica.Address != "bananaphotos-one" {
		t.Fatalf("AskForSearchDocuments = %+v, want the first replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "bananaphotos-one")
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByCoverage)
	asking.observer.wantAmountOfDocumentsListed(t, 0)
}

func TestAFailureAsksTheNextReplicaAtOnce(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {fails: true}, "berlin-two": {documentsListed: 2},
	}, 1)

	answers := asking.answersOf(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 1 || answers[0].Replica.Address != "berlin-two" {
		t.Fatalf("the run = %+v, want the second replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.observer.wantCoveringAskPutOn(t, wordpartitionasks.PutOnFailure)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart, wordpartitionasks.PutOnFailure)
}

func TestACallPastTheHedgeDelayAsksTheNextReplicaAndTheFirstListingWins(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 3, answersWhenReleased: true},
		"berlin-two": {documentsListed: 2},
	}, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one", "berlin-two")
	asking.calls.waitForThePutOf("berlin-one")
	asking.clock.nextTimer(t).expire()
	settledAsk := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, settledAsk, "berlin-two")
	asking.calls.wantAddressesCancelled(t, "berlin-one")
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.clock.wantTimersStopped(t, 2)
	asking.observer.wantCoveringAskPutOn(t, wordpartitionasks.PutOnHedgeDelay)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart, wordpartitionasks.PutOnHedgeDelay)
}

func TestAHedgeIsDueWhileTheFirstReplicaStillHoldsTheCall(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 3, answersWhenReleased: true},
		"berlin-two": {documentsListed: 2, answersWhenReleased: true},
	}, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one", "berlin-two")
	asking.clock.nextTimer(t).expire()
	asking.calls.waitForThePutOf("berlin-two")
	asking.calls.release("berlin-one")
	settledAsk := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, settledAsk, "berlin-one")
	asking.calls.wantAddressesCancelled(t, "berlin-two")
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.observer.wantCoveringAskPutOn(t, wordpartitionasks.PutOnStart)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart, wordpartitionasks.PutOnHedgeDelay)
}

func TestAHedgeDueForAnEndedCallAsksNoReplica(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one":   {documentsListed: 0},
		"berlin-two":   {documentsListed: 2, answersWhenReleased: true},
		"berlin-three": {documentsListed: 1, answersWhenReleased: true},
		"berlin-four":  {documentsListed: 4},
	}, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord(
		"berlin", 1, "berlin-one", "berlin-two", "berlin-three", "berlin-four",
	)
	timerOfTheEndedCall := asking.clock.nextTimer(t)
	timerOfTheHeldCall := asking.clock.nextTimer(t)
	timerOfTheEndedCall.expire()
	timerOfTheHeldCall.expire()
	asking.calls.waitForThePutOf("berlin-three")
	asking.calls.release("berlin-two")
	settledAsk := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, settledAsk, "berlin-one", "berlin-two")
	asking.calls.wantAddressesCancelled(t, "berlin-three")
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two", "berlin-three")
	asking.observer.wantPutOn(
		t,
		wordpartitionasks.PutOnStart,
		wordpartitionasks.PutOnEmptyAnswer,
		wordpartitionasks.PutOnHedgeDelay,
	)
}

func TestNoReplicaLeftSettlesTheWordPartition(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 3}, "berlin-two": {documentsListed: 0},
	}, 2)

	answers := asking.answersOf(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 2 {
		t.Fatalf(
			"the run answered %d asks, want both answers kept",
			len(answers),
		)
	}
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByNoReplicaLeft)
	asking.observer.wantCoveringAskPutOn(t, "")
	asking.observer.wantAmountOfDocumentsListed(t, 3)
}

func TestAsksPerWordPartitionNeverExceedTheReplicasGiven(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {fails: true}, "berlin-two": {fails: true},
	}, 2)

	answers := asking.answersOf(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 0 {
		t.Fatalf("the run = %+v, want no answer", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByNoReplicaLeft)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart, wordpartitionasks.PutOnStart)
}

func TestTheDeadlineOfTheAsksSettlesTheWordPartitionAndKeepsItsAnswers(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one":  {documentsListed: 3},
		"weather-one": {documentsListed: 2, answersWhenReleased: true},
	}, 1)
	ctx, endTheAsks := context.WithCancel(t.Context())
	defer endTheAsks()
	run := asking.startedRun(ctx)

	run.Asks <- append(
		asksForTheWord("berlin", 1, "berlin-one"),
		asksForTheWord("weather", 2, "weather-one")...,
	)
	coveredAsk := <-run.SettledAsks
	endTheAsks()
	askAtTheDeadline := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, coveredAsk, "berlin-one")
	wantAnswersIn(t, askAtTheDeadline)
	asking.observer.wantSettledBy(
		t,
		wordpartitionasks.SettledByCoverage,
		wordpartitionasks.SettledByDeadline,
	)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByDeadline)
}

func TestAFailureThatArrivesAfterTheDeadlineSettlesTheWordPartitionAsDeadline(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 3, answersWhenReleased: true},
		"berlin-two": {documentsListed: 2},
	}, 1)
	ctx, endTheAsks := context.WithCancel(t.Context())
	endTheAsks()
	run := asking.startedRun(ctx)

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one", "berlin-two")
	asking.calls.release("berlin-one")
	settledAsk := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, settledAsk)
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByDeadline)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByDeadline)
}

func TestEachSettledAskTellsTheAnswersOfItsReplicas(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"seven-one": {documentsListed: 1}, "seven-two": {documentsListed: 1},
		"three-one": {documentsListed: 2},
	}, 1)

	answers := asking.answersOf(t.Context(), append(
		asksForTheWord("berlin", 7, "seven-one", "seven-two"),
		asksForTheWord("berlin", 3, "three-one")...,
	))

	wanted := []string{"three-one", "seven-one"}
	if got := addressesOf(answers); !slices.Equal(got, wanted) {
		t.Fatalf("the run answered from %v, want %v", got, wanted)
	}
	asking.calls.wantAddressesPut(t, "three-one", "seven-one")
	asking.observer.wantAmountOfDocumentsListed(t, 1, 2)
}

func TestTheSettledAsksHoldOnlyTheAnswersThatCameBack(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {fails: true}, "berlin-two": {documentsListed: 1},
		"weather-one": {documentsListed: 1}, "weather-two": {documentsListed: 1},
	}, 1)

	answers := asking.answersOf(t.Context(), append(
		asksForTheWord("berlin", 1, "berlin-one", "berlin-two", "berlin-three"),
		asksForTheWord("weather", 2, "weather-one", "weather-two")...,
	))

	wanted := []string{"berlin-two", "weather-one"}
	if got := addressesOf(answers); !slices.Equal(got, wanted) {
		t.Fatalf("the run answered from %v, want %v", got, wanted)
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two", "weather-one")
}

func TestAnEmptyAnswerThatCountsDocumentsHeldSettlesTheWordPartition(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsHeld: yacymodel.Some(5)},
		"berlin-two": {documentsListed: 1},
	}, 1)

	answers := asking.answersOf(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if got := addressesOf(answers); !slices.Equal(got, []string{"berlin-one"}) {
		t.Fatalf("the run answered from %v, want the first replica only", got)
	}
	asking.calls.wantAddressesPut(t, "berlin-one")
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByCoverage)
	asking.observer.wantAmountOfDocumentsListed(t, 5)
}

func TestAPeerAskedForOneWordPartitionIsNotAskedForAnotherOne(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"shared": {documentsListed: 3}, "berlin-two": {documentsListed: 2},
		"weather-two": {documentsListed: 4},
	}, 1)

	answers := asking.answersOf(t.Context(), append(
		asksForTheWord("berlin", 1, "shared", "berlin-two"),
		asksForTheWord("weather", 2, "shared", "weather-two")...,
	))

	wanted := []string{"shared", "weather-two"}
	if got := addressesOf(answers); !slices.Equal(got, wanted) {
		t.Fatalf("the run answered from %v, want %v", got, wanted)
	}
	asking.calls.wantAddressesPut(t, "shared", "weather-two")
	asking.observer.wantCoveringAskPutOn(
		t,
		wordpartitionasks.PutOnStart,
		wordpartitionasks.PutOnStart,
	)
}

func TestAWordPartitionWhoseReplicasWereAllAskedForOthersSettlesWithNoReplicaLeft(
	t *testing.T,
) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"shared": {documentsListed: 3},
	}, 1)

	answers := asking.answersOf(t.Context(), append(
		asksForTheWord("berlin", 1, "shared"),
		asksForTheWord("weather", 2, "shared")...,
	))

	if got := addressesOf(answers); !slices.Equal(got, []string{"shared"}) {
		t.Fatalf("the run answered from %v, want the shared peer once", got)
	}
	asking.calls.wantAddressesPut(t, "shared")
	asking.observer.wantSettledBy(
		t, wordpartitionasks.SettledByCoverage, wordpartitionasks.SettledByNoReplicaLeft,
	)
}

func TestAsksAddedToARunningRunAreAsked(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 3}, "weather-one": {documentsListed: 2},
	}, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one")
	firstAsk := <-run.SettledAsks
	run.Asks <- asksForTheWord("weather", 2, "weather-one")
	secondAsk := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, firstAsk, "berlin-one")
	wantAnswersIn(t, secondAsk, "weather-one")
	asking.calls.wantAddressesPut(t, "berlin-one", "weather-one")
	asking.observer.wantSettledBy(
		t,
		wordpartitionasks.SettledByCoverage,
		wordpartitionasks.SettledByCoverage,
	)
}

func TestAPeerAskedEarlierInTheRunIsNotAskedForAnAskAddedLater(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"shared": {documentsListed: 3}, "weather-two": {documentsListed: 2},
	}, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord("berlin", 1, "shared")
	<-run.SettledAsks
	run.Asks <- asksForTheWord("weather", 2, "shared", "weather-two")
	laterAsk := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, laterAsk, "weather-two")
	asking.calls.wantAddressesPut(t, "shared", "weather-two")
}

func TestAWordPartitionAlreadyInTheRunIsNotAskedAgain(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 0}, "berlin-two": {documentsListed: 2},
	}, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one")
	<-run.SettledAsks
	run.Asks <- asksForTheWord("berlin", 1, "berlin-two")
	wantTheRunOver(t, run)

	asking.calls.wantAddressesPut(t, "berlin-one")
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByNoReplicaLeft)
}

func TestEachWordPartitionIsSentAsSoonAsItSettles(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"weather-one": {documentsListed: 2, answersWhenReleased: true},
		"berlin-one":  {documentsListed: 3},
	}, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- append(
		asksForTheWord("weather", 2, "weather-one"),
		asksForTheWord("berlin", 1, "berlin-one")...,
	)
	askBeforeTheRelease := <-run.SettledAsks
	asking.calls.release("weather-one")
	askAfterTheRelease := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, askBeforeTheRelease, "berlin-one")
	wantAnswersIn(t, askAfterTheRelease, "weather-one")
}

func TestTheDeadlineEndsTheRunAndSendsTheUnsettledWordPartitions(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{
		"berlin-one": {documentsListed: 3, answersWhenReleased: true},
	}, 1)
	ctx, endTheRun := context.WithCancel(t.Context())
	defer endTheRun()
	run := asking.startedRun(ctx)

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one")
	endTheRun()
	askAtTheDeadline := <-run.SettledAsks
	wantTheRunOver(t, run)

	wantAnswersIn(t, askAtTheDeadline)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart)
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByDeadline)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByDeadline)
}

func TestARunWithoutAsksIsOverOnceTheAsksClose(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(t, map[string]scriptedReplicaCall{}, 1)
	run := asking.startedRun(t.Context())

	wantTheRunOver(t, run)

	asking.observer.wantSettledBy(t)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByCoverage)
}

type scriptedReplicaCall struct {
	documentsListed     int
	documentsHeld       yacymodel.Optional[int]
	searched            bool
	fails               bool
	answersWhenReleased bool
}

type replicaCallsOfTheTests struct {
	scripts      map[string]scriptedReplicaCall
	heldCalls    map[string]*heldPeerCall
	cancels      chan string
	mutex        sync.Mutex
	addressesPut []string
}

func replicaCallsOf(t *testing.T, scripts map[string]scriptedReplicaCall) *replicaCallsOfTheTests {
	t.Helper()

	calls := &replicaCallsOfTheTests{
		scripts:   scripts,
		heldCalls: map[string]*heldPeerCall{},
		cancels:   make(chan string, amountOfCancelsBuffered),
	}
	for address, script := range scripts {
		if script.answersWhenReleased {
			calls.heldCalls[address] = &heldPeerCall{
				put:      make(chan struct{}),
				released: make(chan struct{}),
			}
		}
	}
	t.Cleanup(calls.releaseTheHeldCalls)

	return calls
}

func (calls *replicaCallsOfTheTests) releaseTheHeldCalls() {
	for address := range calls.heldCalls {
		calls.release(address)
	}
}

func (calls *replicaCallsOfTheTests) release(address string) {
	calls.heldCalls[address].release()
}

func (calls *replicaCallsOfTheTests) Put(
	ctx context.Context,
	_ wordpartitionasks.Ask,
	replica peerdirectory.AskablePeer,
) (wordpartitionasks.ReplicaAnswer, bool) {
	script, answered := calls.answered(ctx, replica.Address)
	if !answered {
		return wordpartitionasks.ReplicaAnswer{}, false
	}

	return wordpartitionasks.ReplicaAnswer{
		Replica:               replica,
		ListedDocuments:       make([]wordpartitionasks.ListedDocument, script.documentsListed),
		AmountOfDocumentsHeld: script.documentsHeld,
		Searched:              script.searched,
	}, true
}

func (calls *replicaCallsOfTheTests) answered(
	ctx context.Context,
	address string,
) (scriptedReplicaCall, bool) {
	script := calls.recordThePut(address)
	if script.answersWhenReleased && !calls.releasedBeforeTheCancel(ctx, address) {
		return script, false
	}

	return script, !script.fails
}

func (calls *replicaCallsOfTheTests) recordThePut(address string) scriptedReplicaCall {
	calls.mutex.Lock()
	defer calls.mutex.Unlock()

	calls.addressesPut = append(calls.addressesPut, address)

	return calls.scripts[address]
}

func (calls *replicaCallsOfTheTests) releasedBeforeTheCancel(
	ctx context.Context,
	address string,
) bool {
	heldCall := calls.heldCalls[address]
	close(heldCall.put)
	select {
	case <-heldCall.released:
		return true
	case <-ctx.Done():
		calls.cancels <- address
		<-heldCall.released

		return false
	}
}

func (calls *replicaCallsOfTheTests) waitForThePutOf(address string) {
	<-calls.heldCalls[address].put
}

func (calls *replicaCallsOfTheTests) wantAddressesPut(t *testing.T, addresses ...string) {
	t.Helper()
	calls.mutex.Lock()
	defer calls.mutex.Unlock()

	if !slices.Equal(slices.Sorted(slices.Values(calls.addressesPut)), slices.Sorted(
		slices.Values(addresses),
	)) {
		t.Fatalf("the replicas asked were %v, want %v", calls.addressesPut, addresses)
	}
}

func (calls *replicaCallsOfTheTests) wantAddressesCancelled(t *testing.T, addresses ...string) {
	t.Helper()

	addressesCancelled := make([]string, 0, len(addresses))
	for range addresses {
		addressesCancelled = append(addressesCancelled, <-calls.cancels)
	}
	if !slices.Equal(slices.Sorted(slices.Values(addressesCancelled)), slices.Sorted(
		slices.Values(addresses),
	)) {
		t.Fatalf("the calls cancelled were %v, want %v", addressesCancelled, addresses)
	}
}

type heldPeerCall struct {
	put         chan struct{}
	released    chan struct{}
	releaseOnce sync.Once
}

func (held *heldPeerCall) release() {
	held.releaseOnce.Do(func() { close(held.released) })
}

type timerTheTestFires struct {
	timeout time.Duration
	expire  func()
}

type clockTheTestFires struct {
	started chan timerTheTestFires
	stopped chan struct{}
}

func newClockTheTestFires() *clockTheTestFires {
	return &clockTheTestFires{
		started: make(chan timerTheTestFires, amountOfTimersBuffered),
		stopped: make(chan struct{}, amountOfTimersBuffered),
	}
}

func (clock *clockTheTestFires) After(timeout time.Duration, expire func()) func() {
	clock.started <- timerTheTestFires{timeout: timeout, expire: expire}

	return func() { clock.stopped <- struct{}{} }
}

func (clock *clockTheTestFires) nextTimer(t *testing.T) timerTheTestFires {
	t.Helper()

	timer := <-clock.started
	if timer.timeout != hedgeDelayOfTheTests {
		t.Fatalf("the timer waits %v, want the hedge delay %v", timer.timeout, hedgeDelayOfTheTests)
	}

	return timer
}

func (clock *clockTheTestFires) wantTimersStopped(t *testing.T, amountOfTimers int) {
	t.Helper()

	if len(clock.stopped) != amountOfTimers {
		t.Fatalf("%d timers stopped, want %d", len(clock.stopped), amountOfTimers)
	}
}

type recordedReplicaAsks struct {
	mutex           sync.Mutex
	amountOfReports int
	performed       wordpartitionasks.PerformedReplicaAsks
}

func (recorded *recordedReplicaAsks) ReplicaAsksPerformed(
	_ context.Context,
	replicaAsks wordpartitionasks.PerformedReplicaAsks,
) {
	recorded.mutex.Lock()
	defer recorded.mutex.Unlock()

	recorded.amountOfReports++
	recorded.performed = replicaAsks
}

func (recorded *recordedReplicaAsks) wantSettledBy(
	t *testing.T,
	settledBy ...wordpartitionasks.SettledBy,
) {
	t.Helper()
	wordPartitions := recorded.wordPartitions(t)
	settledByReported := make([]wordpartitionasks.SettledBy, 0, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		settledByReported = append(settledByReported, wordPartition.SettledBy)
	}
	if !slices.Equal(settledByReported, settledBy) {
		t.Fatalf("the word partitions settled by %q, want %q", settledByReported, settledBy)
	}
}

func (recorded *recordedReplicaAsks) wantCoveringAskPutOn(
	t *testing.T,
	putOn ...wordpartitionasks.PutOn,
) {
	t.Helper()
	wordPartitions := recorded.wordPartitions(t)
	putOnReported := make([]wordpartitionasks.PutOn, 0, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		putOnReported = append(putOnReported, wordPartition.CoveringAskPutOn)
	}
	if !slices.Equal(putOnReported, putOn) {
		t.Fatalf("the covering asks were put on %q, want %q", putOnReported, putOn)
	}
}

func (recorded *recordedReplicaAsks) wantPutOn(t *testing.T, putOn ...wordpartitionasks.PutOn) {
	t.Helper()
	putOnReported := recorded.wordPartitions(t)[0].AsksPutOn
	if !slices.Equal(putOnReported, putOn) {
		t.Fatalf("the replica asks were put on %q, want %q", putOnReported, putOn)
	}
}

func (recorded *recordedReplicaAsks) wantAmountOfDocumentsListed(t *testing.T, amounts ...int) {
	t.Helper()
	wordPartitions := recorded.wordPartitions(t)
	for place, amount := range amounts {
		if wordPartitions[place].AmountOfDocumentsListed != amount {
			t.Fatalf(
				"word partition %d listed %d documents, want %d",
				place, wordPartitions[place].AmountOfDocumentsListed, amount,
			)
		}
	}
}

func (recorded *recordedReplicaAsks) wantEndedBy(t *testing.T, endedBy wordpartitionasks.EndedBy) {
	t.Helper()
	recorded.mutex.Lock()
	defer recorded.mutex.Unlock()

	if recorded.performed.EndedBy != endedBy {
		t.Fatalf("the replica asks ended by %q, want %q", recorded.performed.EndedBy, endedBy)
	}
	if recorded.performed.TimeSpent <= 0 {
		t.Fatalf("the replica asks spent %v, want the time they took", recorded.performed.TimeSpent)
	}
}

func (recorded *recordedReplicaAsks) wordPartitions(
	t *testing.T,
) []wordpartitionasks.PerformedWordPartition {
	t.Helper()
	recorded.mutex.Lock()
	defer recorded.mutex.Unlock()

	if recorded.amountOfReports != 1 {
		t.Fatalf("the observer was told %d times, want once", recorded.amountOfReports)
	}

	return recorded.performed.WordPartitions
}

type hedgeDelayOfTheTestsPeers time.Duration

func (hedgeDelay hedgeDelayOfTheTestsPeers) HedgeDelayOf(
	_ context.Context,
	_ peerdirectory.AskablePeer,
) time.Duration {
	return time.Duration(hedgeDelay)
}

type askingUnderTest struct {
	calls    *replicaCallsOfTheTests
	clock    *clockTheTestFires
	observer *recordedReplicaAsks
	asks     wordpartitionasks.Asks
}

func askingOfTheTests(
	t *testing.T,
	scripts map[string]scriptedReplicaCall,
	amountOfReplicasCoveringAPartition int,
) askingUnderTest {
	t.Helper()

	calls := replicaCallsOf(t, scripts)
	clock := newClockTheTestFires()
	observer := &recordedReplicaAsks{}

	return askingUnderTest{
		calls:    calls,
		clock:    clock,
		observer: observer,
		asks: wordpartitionasks.New(
			calls,
			hedgeDelayOfTheTestsPeers(hedgeDelayOfTheTests),
			clock,
			amountOfReplicasCoveringAPartition,
			observer,
		),
	}
}

func (asking askingUnderTest) answersOf(
	ctx context.Context,
	asks []wordpartitionasks.Ask,
) []wordpartitionasks.ReplicaAnswer {
	run := asking.startedRun(ctx)
	run.Asks <- asks
	close(run.Asks)
	var settledAsks []wordpartitionasks.SettledAsk
	for settledAsk := range run.SettledAsks {
		settledAsks = append(settledAsks, settledAsk)
	}
	slices.SortStableFunc(settledAsks, func(first, second wordpartitionasks.SettledAsk) int {
		return cmp.Compare(first.Partition, second.Partition)
	})
	var answers []wordpartitionasks.ReplicaAnswer
	for _, settledAsk := range settledAsks {
		answers = append(answers, settledAsk.Answers...)
	}

	return answers
}

func (asking askingUnderTest) startedRun(ctx context.Context) wordpartitionasks.Run {
	return asking.asks.Start(ctx)
}

func wantAnswersIn(
	t *testing.T,
	settledAsk wordpartitionasks.SettledAsk,
	addresses ...string,
) {
	t.Helper()

	if got := addressesOf(settledAsk.Answers); !slices.Equal(got, addresses) {
		t.Fatalf("the settled ask holds answers from %v, want %v", got, addresses)
	}
}

func addressesOf(answers []wordpartitionasks.ReplicaAnswer) []string {
	addresses := make([]string, 0, len(answers))
	for _, answer := range answers {
		addresses = append(addresses, answer.Replica.Address)
	}

	return addresses
}

func wantTheRunOver(t *testing.T, run wordpartitionasks.Run) {
	t.Helper()

	close(run.Asks)
	for settledAsk := range run.SettledAsks {
		t.Fatalf(
			"the run sent answers from %v after its last ask, want none",
			addressesOf(settledAsk.Answers),
		)
	}
}

func asksForTheWord(
	word string,
	partition uint,
	addresses ...string,
) []wordpartitionasks.Ask {
	replicas := make([]peerdirectory.AskablePeer, 0, len(addresses))
	for _, address := range addresses {
		replicas = append(replicas, peerAt(address))
	}

	return []wordpartitionasks.Ask{{
		Word:            yacymodel.WordHash(word),
		Partition:       partition,
		ReplicasInOrder: replicas,
	}}
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address}
}
