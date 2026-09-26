package wordpartitionasks_test

import (
	"cmp"
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	hedgeDelayOfTheTests     = 40 * time.Millisecond
	noHedgeOfTheTests        = time.Hour
	slowAnswerOfTheTests     = 400 * time.Millisecond
	deadlineOfTheTests       = 120 * time.Millisecond
	answerAfterTheHedgeIsDue = 200 * time.Millisecond
)

func TestTheFirstReplicasOfEveryWordPartitionAreAskedAndNoMore(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3}, "berlin-two": {documentsListed: 2},
		"berlin-three": {documentsListed: 1}, "weather-one": {documentsListed: 4},
		"weather-two": {documentsListed: 5}, "weather-three": {documentsListed: 6},
	}, noHedgeOfTheTests, 2)

	answers := asking.searchDocumentsAnswers(t.Context(), append(
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

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3}, "berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	answers := asking.searchDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "berlin-one" {
		t.Fatalf("the run = %+v, want the first replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one")
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByCoverage)
	asking.observer.wantCoveringAskPutOn(t, wordpartitionasks.PutOnStart)
	asking.observer.wantAmountOfDocumentsListed(t, 3)
}

func TestAnEmptyAnswerWithoutASearchAsksTheNextReplicaAtOnce(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 0}, "berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	answers := asking.searchDocumentsAnswers(
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

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"bananaphotos-one": {searched: true}, "bananaphotos-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	answers := asking.searchDocumentsAnswers(
		t.Context(), asksForTheWord("bananaphotos", 1, "bananaphotos-one", "bananaphotos-two"),
	)

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "bananaphotos-one" {
		t.Fatalf("AskForSearchDocuments = %+v, want the first replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "bananaphotos-one")
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByCoverage)
	asking.observer.wantAmountOfDocumentsListed(t, 0)
}

func TestAFailureAsksTheNextReplicaAtOnce(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {fails: true}, "berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	answers := asking.searchDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "berlin-two" {
		t.Fatalf("the run = %+v, want the second replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.observer.wantCoveringAskPutOn(t, wordpartitionasks.PutOnFailure)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart, wordpartitionasks.PutOnFailure)
}

func TestACallPastTheHedgeDelayAsksTheNextReplicaAndTheFirstListingWins(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3, answersAfter: slowAnswerOfTheTests},
		"berlin-two": {documentsListed: 2},
	}, hedgeDelayOfTheTests, 1)

	answers := asking.searchDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "berlin-two" {
		t.Fatalf("the run = %+v, want the hedged replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.calls.wantAddressesCancelled(t, "berlin-one")
	asking.observer.wantCoveringAskPutOn(t, wordpartitionasks.PutOnHedgeDelay)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart, wordpartitionasks.PutOnHedgeDelay)
}

func TestAHedgeIsDueWhileTheFirstReplicaStillHoldsTheCall(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3, answersAfter: answerAfterTheHedgeIsDue},
		"berlin-two": {documentsListed: 2, answersAfter: slowAnswerOfTheTests},
	}, hedgeDelayOfTheTests, 1)

	asking.searchDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	putAfter := asking.calls.putAfterOf(t, "berlin-two")
	if putAfter < hedgeDelayOfTheTests || putAfter >= answerAfterTheHedgeIsDue {
		t.Fatalf(
			"the second replica was asked %v after the start, want between %v and %v",
			putAfter, hedgeDelayOfTheTests, answerAfterTheHedgeIsDue,
		)
	}
}

func TestNoReplicaLeftSettlesTheWordPartition(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3}, "berlin-two": {documentsListed: 0},
	}, noHedgeOfTheTests, 2)

	answers := asking.searchDocumentsAnswers(
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

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {fails: true}, "berlin-two": {fails: true},
	}, noHedgeOfTheTests, 2)

	answers := asking.searchDocumentsAnswers(
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

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one":  {documentsListed: 3},
		"weather-one": {documentsListed: 2, answersAfter: slowAnswerOfTheTests},
	}, noHedgeOfTheTests, 1)
	ctx, endTheAsks := context.WithTimeout(t.Context(), deadlineOfTheTests)
	defer endTheAsks()

	answers := asking.searchDocumentsAnswers(ctx, append(
		asksForTheWord("berlin", 1, "berlin-one"),
		asksForTheWord("weather", 2, "weather-one")...,
	))

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "berlin-one" {
		t.Fatalf("the run = %+v, want the answer that came in time", answers)
	}
	asking.observer.wantSettledBy(
		t,
		wordpartitionasks.SettledByCoverage,
		wordpartitionasks.SettledByDeadline,
	)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByDeadline)
}

func TestAFailureThatArrivesAfterTheDeadlineSettlesTheWordPartitionAsDeadline(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3, answersAfter: slowAnswerOfTheTests},
		"berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)
	ctx, endTheAsks := context.WithCancel(t.Context())
	endTheAsks()

	answers := asking.searchDocumentsAnswers(
		ctx, asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 0 {
		t.Fatalf("the run = %+v, want no answer", answers)
	}
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByDeadline)
	asking.observer.wantPutOn(t, wordpartitionasks.PutOnStart)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByDeadline)
}

func TestEachWordPartitionTellsTheOutcomesOfItsAsksInReplicaOrder(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"seven-one": {documentsListed: 1}, "seven-two": {documentsListed: 1},
		"three-one": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	askOutcomes := asking.searchDocumentsAskOutcomes(t.Context(), append(
		asksForTheWord("berlin", 7, "seven-one", "seven-two"),
		asksForTheWord("berlin", 3, "three-one")...,
	))

	wanted := []string{"three-one answered", "seven-one answered", "seven-two not put"}
	if got := outcomesOf(askOutcomes); !slices.Equal(got, wanted) {
		t.Fatalf("the run = %v, want %v", got, wanted)
	}
	asking.observer.wantAmountOfDocumentsListed(t, 1, 2)
}

func TestTheOutcomesTellWhichAsksWerePutAndWhichWereAnswered(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {fails: true}, "berlin-two": {documentsListed: 1},
		"weather-one": {documentsListed: 1}, "weather-two": {documentsListed: 1},
	}, noHedgeOfTheTests, 1)

	askOutcomes := asking.searchDocumentsAskOutcomes(t.Context(), append(
		asksForTheWord("berlin", 1, "berlin-one", "berlin-two", "berlin-three"),
		asksForTheWord("weather", 2, "weather-one", "weather-two")...,
	))

	wanted := []string{
		"berlin-one put", "berlin-two answered", "berlin-three not put",
		"weather-one answered", "weather-two not put",
	}
	if got := outcomesOf(askOutcomes); !slices.Equal(got, wanted) {
		t.Fatalf("the run = %v, want %v", got, wanted)
	}
	if got := addressesOf(askOutcomes.AsksPut()); !slices.Equal(
		got, []string{"berlin-one", "berlin-two", "weather-one"},
	) {
		t.Fatalf("AsksPut = %v, want the three asks put", got)
	}
	if len(askOutcomes.AnsweredAsks()) != 2 {
		t.Fatalf("AnsweredAsks = %+v, want the two that answered", askOutcomes.AnsweredAsks())
	}
}

func TestAnEmptyAnswerThatCountsDocumentsHeldSettlesTheWordPartition(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsHeld: yacymodel.Some(5)},
		"berlin-two": {documentsListed: 1},
	}, noHedgeOfTheTests, 1)

	askOutcomes := asking.searchDocumentsAskOutcomes(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if !slices.Equal(addressesOf(askOutcomes.AsksPut()), []string{"berlin-one"}) ||
		len(askOutcomes.AnsweredAsks()) != 1 {
		t.Fatalf("the run = %+v, want the first replica only", askOutcomes)
	}
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByCoverage)
}

func TestAnAnswerThatOnlyMatchesDocumentsSettlesTheWordPartition(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsMatched: 2},
		"berlin-two": {documentsListed: 1},
	}, noHedgeOfTheTests, 1)

	askOutcomes := asking.searchDocumentsAskOutcomes(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if !slices.Equal(addressesOf(askOutcomes.AsksPut()), []string{"berlin-one"}) ||
		len(askOutcomes.AnsweredAsks()) != 1 {
		t.Fatalf("the run = %+v, want the first replica only", askOutcomes)
	}
	asking.observer.wantAmountOfDocumentsListed(t, 2)
}

func TestAPeerAskedForOneWordPartitionIsNotAskedForAnotherOne(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"shared": {documentsListed: 3}, "berlin-two": {documentsListed: 2},
		"weather-two": {documentsListed: 4},
	}, noHedgeOfTheTests, 1)

	askOutcomes := asking.searchDocumentsAskOutcomes(t.Context(), append(
		asksForTheWord("berlin", 1, "shared", "berlin-two"),
		asksForTheWord("weather", 2, "shared", "weather-two")...,
	))

	wanted := []string{
		"shared answered", "berlin-two not put", "shared not put", "weather-two answered",
	}
	if got := outcomesOf(askOutcomes); !slices.Equal(got, wanted) {
		t.Fatalf("the run = %v, want %v", got, wanted)
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

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"shared": {documentsListed: 3},
	}, noHedgeOfTheTests, 1)

	askOutcomes := asking.searchDocumentsAskOutcomes(t.Context(), append(
		asksForTheWord("berlin", 1, "shared"),
		asksForTheWord("weather", 2, "shared")...,
	))

	if got := addressesOf(askOutcomes.AsksPut()); !slices.Equal(got, []string{"shared"}) {
		t.Fatalf("the asks put went to %v, want the shared peer once", got)
	}
	asking.calls.wantAddressesPut(t, "shared")
	asking.observer.wantSettledBy(
		t, wordpartitionasks.SettledByCoverage, wordpartitionasks.SettledByNoReplicaLeft,
	)
}

func TestAsksAddedToARunningRunAreAsked(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3}, "weather-one": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one")
	firstWordPartition := <-run.SettledWordPartitions
	run.Asks <- asksForTheWord("weather", 2, "weather-one")
	secondWordPartition := <-run.SettledWordPartitions
	wantTheRunOver(t, run)

	wantOutcomesIn(t, firstWordPartition, "berlin-one answered")
	wantOutcomesIn(t, secondWordPartition, "weather-one answered")
	asking.calls.wantAddressesPut(t, "berlin-one", "weather-one")
	asking.observer.wantSettledBy(
		t,
		wordpartitionasks.SettledByCoverage,
		wordpartitionasks.SettledByCoverage,
	)
}

func TestAPeerAskedEarlierInTheRunIsNotAskedForAnAskAddedLater(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"shared": {documentsListed: 3}, "weather-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord("berlin", 1, "shared")
	<-run.SettledWordPartitions
	run.Asks <- asksForTheWord("weather", 2, "shared", "weather-two")
	laterWordPartition := <-run.SettledWordPartitions
	wantTheRunOver(t, run)

	wantOutcomesIn(t, laterWordPartition, "shared not put", "weather-two answered")
	asking.calls.wantAddressesPut(t, "shared", "weather-two")
}

func TestAWordPartitionAlreadyInTheRunIsNotAskedAgain(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 0}, "berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)
	run := asking.startedRun(t.Context())

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one")
	<-run.SettledWordPartitions
	run.Asks <- asksForTheWord("berlin", 1, "berlin-two")
	wantTheRunOver(t, run)

	asking.calls.wantAddressesPut(t, "berlin-one")
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByNoReplicaLeft)
}

func TestEachWordPartitionIsSentAsSoonAsItSettles(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"weather-one": {documentsListed: 2, answersAfter: slowAnswerOfTheTests},
		"berlin-one":  {documentsListed: 3},
	}, noHedgeOfTheTests, 1)
	run := asking.startedRun(t.Context())
	startedAt := time.Now()

	run.Asks <- append(
		asksForTheWord("weather", 2, "weather-one"),
		asksForTheWord("berlin", 1, "berlin-one")...,
	)
	firstWordPartition := <-run.SettledWordPartitions
	firstSettledAfter := time.Since(startedAt)
	secondWordPartition := <-run.SettledWordPartitions
	wantTheRunOver(t, run)

	wantOutcomesIn(t, firstWordPartition, "berlin-one answered")
	if firstSettledAfter >= slowAnswerOfTheTests {
		t.Fatalf(
			"the first word partition came after %v, want it before the slow one at %v",
			firstSettledAfter, slowAnswerOfTheTests,
		)
	}
	wantOutcomesIn(t, secondWordPartition, "weather-one answered")
}

func TestTheDeadlineEndsTheRunAndSendsTheUnsettledWordPartitions(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3, answersAfter: slowAnswerOfTheTests},
	}, noHedgeOfTheTests, 1)
	ctx, endTheRun := context.WithTimeout(t.Context(), deadlineOfTheTests)
	defer endTheRun()
	run := asking.startedRun(ctx)
	startedAt := time.Now()

	run.Asks <- asksForTheWord("berlin", 1, "berlin-one")
	settledWordPartition := <-run.SettledWordPartitions
	settledAfter := time.Since(startedAt)
	wantTheRunOver(t, run)

	wantOutcomesIn(t, settledWordPartition, "berlin-one put")
	if settledAfter >= slowAnswerOfTheTests {
		t.Fatalf("the word partition came after %v, want it at the deadline", settledAfter)
	}
	asking.observer.wantSettledBy(t, wordpartitionasks.SettledByDeadline)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByDeadline)
}

func TestARunWithoutAsksIsOverOnceTheAsksClose(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{}, noHedgeOfTheTests, 1)
	run := asking.startedRun(t.Context())

	wantTheRunOver(t, run)

	asking.observer.wantSettledBy(t)
	asking.observer.wantEndedBy(t, wordpartitionasks.EndedByCoverage)
}

type scriptedPeerCall struct {
	documentsListed  int
	documentsMatched int
	documentsHeld    yacymodel.Optional[int]
	searched         bool
	fails            bool
	answersAfter     time.Duration
}

type peerCallsOfTheTests struct {
	scripts            map[string]scriptedPeerCall
	startedAt          time.Time
	mutex              sync.Mutex
	addressesPut       []string
	putAfter           map[string]time.Duration
	addressesCancelled []string
}

func (calls *peerCallsOfTheTests) AskForSearchDocuments(
	ctx context.Context,
	asks []peerasks.SearchDocumentsAsk,
) []peerasks.AnsweredSearchDocumentsAsk {
	script, answered := calls.answered(ctx, asks[0].Peer.Address)
	if !answered {
		return nil
	}

	return []peerasks.AnsweredSearchDocumentsAsk{{
		Ask:                             asks[0],
		Abstract:                        make([]yacymodel.URLHash, script.documentsListed),
		MatchedDocuments:                make([]peerasks.MatchedDocument, script.documentsMatched),
		AmountOfDocumentsHeldForTheWord: script.documentsHeld,
		PeerSearched:                    script.searched,
	}}
}

func (calls *peerCallsOfTheTests) answered(
	ctx context.Context,
	address string,
) (scriptedPeerCall, bool) {
	script := calls.recordThePut(address)
	if script.answersAfter > 0 {
		answerTimer := time.NewTimer(script.answersAfter)
		defer answerTimer.Stop()
		select {
		case <-answerTimer.C:
		case <-ctx.Done():
			calls.recordTheCancel(address)

			return script, false
		}
	}

	return script, !script.fails
}

func (calls *peerCallsOfTheTests) recordThePut(address string) scriptedPeerCall {
	calls.mutex.Lock()
	defer calls.mutex.Unlock()

	calls.addressesPut = append(calls.addressesPut, address)
	calls.putAfter[address] = time.Since(calls.startedAt)

	return calls.scripts[address]
}

func (calls *peerCallsOfTheTests) recordTheCancel(address string) {
	calls.mutex.Lock()
	defer calls.mutex.Unlock()

	calls.addressesCancelled = append(calls.addressesCancelled, address)
}

func (calls *peerCallsOfTheTests) wantAddressesPut(t *testing.T, addresses ...string) {
	t.Helper()
	calls.mutex.Lock()
	defer calls.mutex.Unlock()

	if len(calls.addressesPut) != len(addresses) {
		t.Fatalf("the replicas asked were %v, want %v", calls.addressesPut, addresses)
	}
	for _, address := range addresses {
		if _, put := calls.putAfter[address]; !put {
			t.Fatalf("the replicas asked were %v, want %v", calls.addressesPut, addresses)
		}
	}
}

func (calls *peerCallsOfTheTests) wantAddressesCancelled(t *testing.T, addresses ...string) {
	t.Helper()

	for range int(slowAnswerOfTheTests / time.Millisecond) {
		if len(calls.addressesCancelledSoFar()) == len(addresses) {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("the calls cancelled were %v, want %v", calls.addressesCancelledSoFar(), addresses)
}

func (calls *peerCallsOfTheTests) addressesCancelledSoFar() []string {
	calls.mutex.Lock()
	defer calls.mutex.Unlock()

	return append([]string{}, calls.addressesCancelled...)
}

func (calls *peerCallsOfTheTests) putAfterOf(t *testing.T, address string) time.Duration {
	t.Helper()
	calls.mutex.Lock()
	defer calls.mutex.Unlock()

	putAfter, put := calls.putAfter[address]
	if !put {
		t.Fatalf("%s was never asked, want it asked when the hedge fell due", address)
	}

	return putAfter
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
	calls    *peerCallsOfTheTests
	observer *recordedReplicaAsks
	asks     wordpartitionasks.Asks
}

func askingOfTheTests(
	scripts map[string]scriptedPeerCall,
	hedgeDelay time.Duration,
	amountOfReplicasCoveringAPartition int,
) askingUnderTest {
	calls := &peerCallsOfTheTests{
		scripts:   scripts,
		startedAt: time.Now(),
		putAfter:  map[string]time.Duration{},
	}
	observer := &recordedReplicaAsks{}

	return askingUnderTest{
		calls:    calls,
		observer: observer,
		asks: wordpartitionasks.New(
			calls,
			hedgeDelayOfTheTestsPeers(hedgeDelay),
			amountOfReplicasCoveringAPartition,
			observer,
		),
	}
}

func (asking askingUnderTest) searchDocumentsAnswers(
	ctx context.Context,
	asks []peerasks.SearchDocumentsAsk,
) []peerasks.AnsweredSearchDocumentsAsk {
	return asking.searchDocumentsAskOutcomes(ctx, asks).AnsweredAsks()
}

func (asking askingUnderTest) searchDocumentsAskOutcomes(
	ctx context.Context,
	asks []peerasks.SearchDocumentsAsk,
) peerasks.SearchDocumentsAskOutcomes {
	run := asking.startedRun(ctx)
	run.Asks <- asks
	close(run.Asks)
	askOutcomes := peerasks.SearchDocumentsAskOutcomes{}
	for settledWordPartition := range run.SettledWordPartitions {
		askOutcomes = append(askOutcomes, settledWordPartition.AskOutcomes...)
	}
	slices.SortStableFunc(askOutcomes, func(first, second peerasks.SearchDocumentsAskOutcome) int {
		return cmp.Compare(first.Ask.Partition, second.Ask.Partition)
	})

	return askOutcomes
}

func (asking askingUnderTest) startedRun(ctx context.Context) wordpartitionasks.Run {
	asking.calls.startedAt = time.Now()

	return asking.asks.Start(ctx)
}

func wantOutcomesIn(
	t *testing.T,
	settledWordPartition wordpartitionasks.SettledWordPartition,
	outcomes ...string,
) {
	t.Helper()

	if got := outcomesIn(settledWordPartition); !slices.Equal(got, outcomes) {
		t.Fatalf("the settled word partition = %v, want %v", got, outcomes)
	}
}

func outcomesIn(settledWordPartition wordpartitionasks.SettledWordPartition) []string {
	return outcomesOf(settledWordPartition.AskOutcomes)
}

func wantTheRunOver(t *testing.T, run wordpartitionasks.Run) {
	t.Helper()

	close(run.Asks)
	for settledWordPartition := range run.SettledWordPartitions {
		t.Fatalf(
			"the run sent %v after its last word partition, want none",
			outcomesIn(settledWordPartition),
		)
	}
}

func asksForTheWord(
	word string,
	partition uint,
	addresses ...string,
) []peerasks.SearchDocumentsAsk {
	asks := make([]peerasks.SearchDocumentsAsk, 0, len(addresses))
	for _, address := range addresses {
		asks = append(asks, peerasks.SearchDocumentsAsk{
			Peer:      peerAt(address),
			Partition: partition,
			Word:      yacymodel.WordHash(word),
		})
	}

	return asks
}

func addressesOf(asks []peerasks.SearchDocumentsAsk) []string {
	addresses := make([]string, 0, len(asks))
	for _, ask := range asks {
		addresses = append(addresses, ask.Peer.Address)
	}

	return addresses
}

func outcomesOf(askOutcomes peerasks.SearchDocumentsAskOutcomes) []string {
	outcomes := make([]string, 0, len(askOutcomes))
	for _, askOutcome := range askOutcomes {
		switch {
		case askOutcome.Answer.Present():
			outcomes = append(outcomes, askOutcome.Ask.Peer.Address+" answered")
		case askOutcome.Put:
			outcomes = append(outcomes, askOutcome.Ask.Peer.Address+" put")
		default:
			outcomes = append(outcomes, askOutcome.Ask.Peer.Address+" not put")
		}
	}

	return outcomes
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(address), Address: address}
}
