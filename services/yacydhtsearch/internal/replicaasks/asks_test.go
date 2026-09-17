package replicaasks_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
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

	answers := asking.matchedAndHeldDocumentsAnswers(t.Context(), append(
		asksForTheWord("berlin", 1, "berlin-one", "berlin-two", "berlin-three"),
		asksForTheWord("weather", 2, "weather-one", "weather-two", "weather-three")...,
	))

	if len(answers) != 4 {
		t.Fatalf("AskForMatchedAndHeldDocuments answered %d asks, want four", len(answers))
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two", "weather-one", "weather-two")
	asking.observer.wantSettledBy(t, replicaasks.SettledByCoverage, replicaasks.SettledByCoverage)
	asking.observer.wantCoveringAskPutOn(t, replicaasks.PutOnStart, replicaasks.PutOnStart)
	asking.observer.wantEndedBy(t, replicaasks.EndedByCoverage)
}

func TestOneReplicaCoveringAPartitionLeavesTheOtherReplicasUnasked(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3}, "berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	answers := asking.matchedAndHeldDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "berlin-one" {
		t.Fatalf("AskForMatchedAndHeldDocuments = %+v, want the first replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one")
	asking.observer.wantSettledBy(t, replicaasks.SettledByCoverage)
	asking.observer.wantCoveringAskPutOn(t, replicaasks.PutOnStart)
	asking.observer.wantAmountOfDocumentsListed(t, 3)
}

func TestAnEmptyAnswerAsksTheNextReplicaAtOnce(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 0}, "berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	answers := asking.matchedAndHeldDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 2 {
		t.Fatalf("AskForMatchedAndHeldDocuments answered %d asks, want both replicas", len(answers))
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.observer.wantCoveringAskPutOn(t, replicaasks.PutOnEmptyAnswer)
	asking.observer.wantPutOn(t, replicaasks.PutOnStart, replicaasks.PutOnEmptyAnswer)
}

func TestAFailureAsksTheNextReplicaAtOnce(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {fails: true}, "berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	answers := asking.matchedAndHeldDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "berlin-two" {
		t.Fatalf("AskForMatchedAndHeldDocuments = %+v, want the second replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.observer.wantCoveringAskPutOn(t, replicaasks.PutOnFailure)
	asking.observer.wantPutOn(t, replicaasks.PutOnStart, replicaasks.PutOnFailure)
}

func TestACallPastTheHedgeDelayAsksTheNextReplicaAndTheFirstListingWins(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3, answersAfter: slowAnswerOfTheTests},
		"berlin-two": {documentsListed: 2},
	}, hedgeDelayOfTheTests, 1)

	answers := asking.matchedAndHeldDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "berlin-two" {
		t.Fatalf("AskForMatchedAndHeldDocuments = %+v, want the hedged replica only", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.calls.wantAddressesCancelled(t, "berlin-one")
	asking.observer.wantCoveringAskPutOn(t, replicaasks.PutOnHedgeDelay)
	asking.observer.wantPutOn(t, replicaasks.PutOnStart, replicaasks.PutOnHedgeDelay)
}

func TestAHedgeIsDueWhileTheFirstReplicaStillHoldsTheCall(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3, answersAfter: answerAfterTheHedgeIsDue},
		"berlin-two": {documentsListed: 2, answersAfter: slowAnswerOfTheTests},
	}, hedgeDelayOfTheTests, 1)

	asking.matchedAndHeldDocumentsAnswers(
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

	answers := asking.matchedAndHeldDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 2 {
		t.Fatalf(
			"AskForMatchedAndHeldDocuments answered %d asks, want both answers kept",
			len(answers),
		)
	}
	asking.observer.wantSettledBy(t, replicaasks.SettledByNoReplicaLeft)
	asking.observer.wantCoveringAskPutOn(t, "")
	asking.observer.wantAmountOfDocumentsListed(t, 3)
}

func TestAsksPerWordPartitionNeverExceedTheReplicasGiven(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {fails: true}, "berlin-two": {fails: true},
	}, noHedgeOfTheTests, 2)

	answers := asking.matchedAndHeldDocumentsAnswers(
		t.Context(), asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 0 {
		t.Fatalf("AskForMatchedAndHeldDocuments = %+v, want no answer", answers)
	}
	asking.calls.wantAddressesPut(t, "berlin-one", "berlin-two")
	asking.observer.wantSettledBy(t, replicaasks.SettledByNoReplicaLeft)
	asking.observer.wantPutOn(t, replicaasks.PutOnStart, replicaasks.PutOnStart)
}

func TestTheDeadlineOfTheAsksSettlesTheWordPartitionAndKeepsItsAnswers(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one":  {documentsListed: 3},
		"weather-one": {documentsListed: 2, answersAfter: slowAnswerOfTheTests},
	}, noHedgeOfTheTests, 1)
	ctx, endTheAsks := context.WithTimeout(t.Context(), deadlineOfTheTests)
	defer endTheAsks()

	answers := asking.matchedAndHeldDocumentsAnswers(ctx, append(
		asksForTheWord("berlin", 1, "berlin-one"),
		asksForTheWord("weather", 2, "weather-one")...,
	))

	if len(answers) != 1 || answers[0].Ask.Peer.Address != "berlin-one" {
		t.Fatalf("AskForMatchedAndHeldDocuments = %+v, want the answer that came in time", answers)
	}
	asking.observer.wantSettledBy(t, replicaasks.SettledByCoverage, replicaasks.SettledByDeadline)
	asking.observer.wantEndedBy(t, replicaasks.EndedByDeadline)
}

func TestAFailureThatArrivesAfterTheDeadlineSettlesTheWordPartitionAsDeadline(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"berlin-one": {documentsListed: 3, answersAfter: slowAnswerOfTheTests},
		"berlin-two": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)
	ctx, endTheAsks := context.WithCancel(t.Context())
	endTheAsks()

	answers := asking.matchedAndHeldDocumentsAnswers(
		ctx, asksForTheWord("berlin", 1, "berlin-one", "berlin-two"),
	)

	if len(answers) != 0 {
		t.Fatalf("AskForMatchedAndHeldDocuments = %+v, want no answer", answers)
	}
	asking.observer.wantSettledBy(t, replicaasks.SettledByDeadline)
	asking.observer.wantPutOn(t, replicaasks.PutOnStart)
	asking.observer.wantEndedBy(t, replicaasks.EndedByDeadline)
}

func TestTheAnswersComeBackInTheOrderTheWordPartitionsWereFirstAsked(t *testing.T) {
	t.Parallel()

	asking := askingOfTheTests(map[string]scriptedPeerCall{
		"seven-one": {documentsListed: 1}, "three-one": {documentsListed: 2},
	}, noHedgeOfTheTests, 1)

	answers := asking.matchedDocumentsAnswers(t.Context(), []peerasks.MatchedDocumentsAsk{
		{Peer: peerAt("seven-one"), Partition: 7},
		{Peer: peerAt("three-one"), Partition: 3},
	})

	if len(answers) != 2 {
		t.Fatalf("AskForMatchedDocuments answered %d asks, want two", len(answers))
	}
	if answers[0].Ask.Partition != 7 || answers[1].Ask.Partition != 3 {
		t.Fatalf(
			"AskForMatchedDocuments = %+v, want partition seven before partition three",
			answers,
		)
	}
	asking.observer.wantAmountOfDocumentsListed(t, 1, 2)
}

type scriptedPeerCall struct {
	documentsListed int
	fails           bool
	answersAfter    time.Duration
}

type peerCallsOfTheTests struct {
	scripts            map[string]scriptedPeerCall
	startedAt          time.Time
	mutex              sync.Mutex
	addressesPut       []string
	putAfter           map[string]time.Duration
	addressesCancelled []string
}

func (calls *peerCallsOfTheTests) AskForMatchedAndHeldDocuments(
	ctx context.Context,
	asks []peerasks.MatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	script, answered := calls.answered(ctx, asks[0].Peer.Address)
	if !answered {
		return nil
	}

	return []peerasks.AnsweredMatchedAndHeldDocumentsAsk{{
		Ask:                       asks[0],
		DocumentsListedForTheWord: make([]yacymodel.URLHash, script.documentsListed),
	}}
}

func (calls *peerCallsOfTheTests) AskForMatchedDocuments(
	ctx context.Context,
	asks []peerasks.MatchedDocumentsAsk,
) []peerasks.AnsweredMatchedDocumentsAsk {
	script, answered := calls.answered(ctx, asks[0].Peer.Address)
	if !answered {
		return nil
	}

	return []peerasks.AnsweredMatchedDocumentsAsk{{
		Ask:              asks[0],
		MatchedDocuments: make([]peerasks.MatchedDocument, script.documentsListed),
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
	performed       replicaasks.PerformedReplicaAsks
}

func (recorded *recordedReplicaAsks) ReplicaAsksPerformed(
	_ context.Context,
	replicaAsks replicaasks.PerformedReplicaAsks,
) {
	recorded.mutex.Lock()
	defer recorded.mutex.Unlock()

	recorded.amountOfReports++
	recorded.performed = replicaAsks
}

func (recorded *recordedReplicaAsks) wantSettledBy(
	t *testing.T,
	settledBy ...replicaasks.SettledBy,
) {
	t.Helper()
	wordPartitions := recorded.wordPartitions(t)
	settledByReported := make([]replicaasks.SettledBy, 0, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		settledByReported = append(settledByReported, wordPartition.SettledBy)
	}
	if !slices.Equal(settledByReported, settledBy) {
		t.Fatalf("the word partitions settled by %q, want %q", settledByReported, settledBy)
	}
}

func (recorded *recordedReplicaAsks) wantCoveringAskPutOn(
	t *testing.T,
	putOn ...replicaasks.PutOn,
) {
	t.Helper()
	wordPartitions := recorded.wordPartitions(t)
	putOnReported := make([]replicaasks.PutOn, 0, len(wordPartitions))
	for _, wordPartition := range wordPartitions {
		putOnReported = append(putOnReported, wordPartition.CoveringAskPutOn)
	}
	if !slices.Equal(putOnReported, putOn) {
		t.Fatalf("the covering asks were put on %q, want %q", putOnReported, putOn)
	}
}

func (recorded *recordedReplicaAsks) wantPutOn(t *testing.T, putOn ...replicaasks.PutOn) {
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

func (recorded *recordedReplicaAsks) wantEndedBy(t *testing.T, endedBy replicaasks.EndedBy) {
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
) []replicaasks.SettledWordPartition {
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
	asks     replicaasks.Asks
}

func askingOfTheTests(
	scripts map[string]scriptedPeerCall,
	hedgeDelay time.Duration,
	replicasCoveringAPartition int,
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
		asks: replicaasks.New(
			calls,
			hedgeDelayOfTheTestsPeers(hedgeDelay),
			replicasCoveringAPartition,
			observer,
		),
	}
}

func (asking askingUnderTest) matchedAndHeldDocumentsAnswers(
	ctx context.Context,
	asks []peerasks.MatchedAndHeldDocumentsAsk,
) []peerasks.AnsweredMatchedAndHeldDocumentsAsk {
	asking.calls.startedAt = time.Now()

	return asking.asks.AskForMatchedAndHeldDocuments(ctx, asks)
}

func (asking askingUnderTest) matchedDocumentsAnswers(
	ctx context.Context,
	asks []peerasks.MatchedDocumentsAsk,
) []peerasks.AnsweredMatchedDocumentsAsk {
	asking.calls.startedAt = time.Now()

	return asking.asks.AskForMatchedDocuments(ctx, asks)
}

func asksForTheWord(
	word string,
	partition uint,
	addresses ...string,
) []peerasks.MatchedAndHeldDocumentsAsk {
	asks := make([]peerasks.MatchedAndHeldDocumentsAsk, 0, len(addresses))
	for _, address := range addresses {
		asks = append(asks, peerasks.MatchedAndHeldDocumentsAsk{
			Peer:      peerAt(address),
			Partition: partition,
			Word:      yacymodel.WordHash(word),
		})
	}

	return asks
}

func peerAt(address string) peerdirectory.AskablePeer {
	return peerdirectory.AskablePeer{Address: address}
}
