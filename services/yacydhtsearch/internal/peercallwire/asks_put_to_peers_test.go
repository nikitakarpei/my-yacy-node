package peercallwire_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

const (
	shortSpreadBudget   = 500 * time.Millisecond
	shortPeerCallBudget = 500 * time.Millisecond
)

type peerNetwork struct {
	mutex                   sync.Mutex
	callsInFlight           int
	mostCallsInFlight       int
	awaitedCallsInFlight    int
	awaitedCallsAreInFlight chan struct{}
	calledHostsInOrder      []string
}

func (n *peerNetwork) awaitsCallsInFlight(calls int) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.awaitedCallsInFlight = calls
	n.awaitedCallsAreInFlight = make(chan struct{})
}

func (n *peerNetwork) callsSeenInFlightAtOnce() int {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	return n.mostCallsInFlight
}

func (n *peerNetwork) hostsSeenCalledInOrder() []string {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	return slices.Clone(n.calledHostsInOrder)
}

func (n *peerNetwork) peerHolding(t *testing.T, addresses ...string) string {
	t.Helper()

	return n.peerAnsweringAfter(t, 0, searchAnswerHolding(t, addresses...))
}

func (n *peerNetwork) peerAnsweringAfter(t *testing.T, delay time.Duration, body string) string {
	t.Helper()

	return n.peerAnswering(t, body, func(peerCall *http.Request) bool {
		select {
		case <-time.After(delay):
			return true
		case <-peerCall.Context().Done():
			return false
		}
	})
}

func (n *peerNetwork) peerAnsweringWhenTheAwaitedCallsAreInFlight(
	t *testing.T,
	body string,
) string {
	t.Helper()

	return n.peerAnswering(t, body, func(peerCall *http.Request) bool {
		select {
		case <-n.awaitedCallsAreInFlight:
			return true
		case <-peerCall.Context().Done():
			return false
		}
	})
}

func (n *peerNetwork) peerAnswering(
	t *testing.T,
	body string,
	theAnswerIsDue func(peerCall *http.Request) bool,
) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, peerCall *http.Request) {
			n.enterCall(peerCall.Host)
			defer n.leaveCall()

			if !theAnswerIsDue(peerCall) {
				return
			}
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func (n *peerNetwork) enterCall(host string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.calledHostsInOrder = append(n.calledHostsInOrder, host)
	n.callsInFlight++
	n.mostCallsInFlight = max(n.mostCallsInFlight, n.callsInFlight)
	if n.awaitedCallsInFlight != 0 && n.callsInFlight == n.awaitedCallsInFlight {
		close(n.awaitedCallsAreInFlight)
	}
}

func (n *peerNetwork) leaveCall() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.callsInFlight--
}

func asksOfPeersAt(addresses ...string) []peerasks.MatchedDocumentsAsk {
	asks := make([]peerasks.MatchedDocumentsAsk, 0, len(addresses))
	for _, address := range addresses {
		asks = append(asks, peerasks.MatchedDocumentsAsk{Peer: peerAt(address)})
	}

	return asks
}

func TestEveryAskThatCameBackCarriesTheAskItAnswers(t *testing.T) {
	t.Parallel()

	network := &peerNetwork{}
	first := network.peerHolding(t, "https://a.example/")
	second := network.peerHolding(t, "https://b.example/")

	answeredAsks := wireTo(&recordedOutcome{}).AskForMatchedDocuments(
		t.Context(),
		asksOfPeersAt(first, second),
	)

	if len(answeredAsks) != 2 {
		t.Fatalf(
			"AskForMatchedDocuments collected %d answers, want one for each ask",
			len(answeredAsks),
		)
	}
	heldByAddress := map[string]string{
		first:  "https://a.example/",
		second: "https://b.example/",
	}
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.MatchedDocuments) != 1 ||
			answeredAsk.MatchedDocuments[0].Metadata.Address !=
				heldByAddress[answeredAsk.Ask.Peer.Address] {
			t.Fatalf(
				"answered ask %+v does not carry the ask its documents came back for",
				answeredAsk,
			)
		}
	}
}

func TestAPeerThatOutlastsTheTimeTheAskHasLeftIsNoAnswer(t *testing.T) {
	t.Parallel()

	network := &peerNetwork{}
	slow := network.peerAnsweringAfter(t, 10*shortSpreadBudget, searchAnswerHolding(t))
	holder := network.peerHolding(t, "https://a.example/")

	startedAt := time.Now()
	answeredAsks := wireTo(&recordedOutcome{}).
		AskForMatchedDocuments(callWithin(t, shortSpreadBudget), asksOfPeersAt(slow, holder))

	if len(answeredAsks) != 1 ||
		answeredAsks[0].MatchedDocuments[0].Metadata.Address != "https://a.example/" {
		t.Fatalf("AskForMatchedDocuments = %+v, want only the peer inside the budget", answeredAsks)
	}
	if time.Since(startedAt) < shortSpreadBudget {
		t.Fatalf(
			"AskForMatchedDocuments returned after %v, want it to wait out the budget",
			time.Since(startedAt),
		)
	}
}

func TestAPeerThatOutlastsThePeerCallBudgetIsNoAnswer(t *testing.T) {
	t.Parallel()

	network := &peerNetwork{}
	slow := network.peerAnsweringAfter(t, 4*shortPeerCallBudget, searchAnswerHolding(t))
	holder := network.peerHolding(t, "https://a.example/")

	startedAt := time.Now()
	answeredAsks := wireSpendingAtMost(shortPeerCallBudget, &recordedOutcome{}).
		AskForMatchedDocuments(callWithin(t, spreadBudgetOfTheTests), asksOfPeersAt(slow, holder))

	if len(answeredAsks) != 1 ||
		answeredAsks[0].MatchedDocuments[0].Metadata.Address != "https://a.example/" {
		t.Fatalf("AskForMatchedDocuments = %+v, want only the peer inside the budget", answeredAsks)
	}
	if time.Since(startedAt) > 2*shortPeerCallBudget {
		t.Fatalf(
			"AskForMatchedDocuments returned after %v, want it to drop the slow peer at %v",
			time.Since(startedAt),
			shortPeerCallBudget,
		)
	}
}

func TestTheWireHoldsItsCallsInFlightAndPutsTheAsksInTheOrderGiven(t *testing.T) {
	t.Parallel()

	const (
		callsInFlight  = 3
		asksOfTheRound = 12
	)

	network := &peerNetwork{}
	network.awaitsCallsInFlight(callsInFlight)
	asks := make([]peerasks.MatchedDocumentsAsk, 0, asksOfTheRound)
	for range asksOfTheRound {
		asks = append(
			asks,
			peerasks.MatchedDocumentsAsk{
				Peer: peerAt(
					network.peerAnsweringWhenTheAwaitedCallsAreInFlight(
						t, searchAnswerHolding(t),
					),
				),
			},
		)
	}

	answeredAsks := wireHolding(callsInFlight, &recordedOutcome{}).
		AskForMatchedDocuments(callWithin(t, spreadBudgetOfTheTests), asks)

	if len(answeredAsks) != asksOfTheRound {
		t.Fatalf(
			"%d asks came back, want all %d: the waiting asks were dropped",
			len(answeredAsks),
			asksOfTheRound,
		)
	}
	if network.callsSeenInFlightAtOnce() != callsInFlight {
		t.Fatalf(
			"%d peer calls were in flight at once, want %d",
			network.callsSeenInFlightAtOnce(),
			callsInFlight,
		)
	}
	firstWave := askIndicesOf(network.hostsSeenCalledInOrder()[:callsInFlight], asks)
	slices.Sort(firstWave)
	if !slices.Equal(firstWave, []int{0, 1, 2}) {
		t.Fatalf("asks %v were put first, want the first %d asks given", firstWave, callsInFlight)
	}
}

func askIndicesOf(hosts []string, asks []peerasks.MatchedDocumentsAsk) []int {
	indexPerHost := map[string]int{}
	for index, ask := range asks {
		indexPerHost[strings.TrimPrefix(ask.Peer.Address, "http://")] = index
	}

	indices := make([]int, 0, len(hosts))
	for _, host := range hosts {
		indices = append(indices, indexPerHost[host])
	}

	return indices
}

func TestAskingNoPeerCollectsNoAnswer(t *testing.T) {
	t.Parallel()

	answeredAsks := wireTo(&recordedOutcome{}).AskForMatchedDocuments(t.Context(), nil)

	if len(answeredAsks) != 0 {
		t.Fatalf("AskForMatchedDocuments = %+v, want no answers", answeredAsks)
	}
}
