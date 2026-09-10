package peercallwire_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
)

const shortCallBudget = 500 * time.Millisecond

type peerNetwork struct {
	mutex                sync.Mutex
	callsInFlight        int
	callsOfTheRound      int
	wholeRoundIsInFlight chan struct{}
}

func (n *peerNetwork) expectsARoundOf(calls int) {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.callsOfTheRound = calls
	n.wholeRoundIsInFlight = make(chan struct{})
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

func (n *peerNetwork) peerAnsweringWhenTheWholeRoundIsInFlight(
	t *testing.T,
	body string,
) string {
	t.Helper()

	return n.peerAnswering(t, body, func(peerCall *http.Request) bool {
		select {
		case <-n.wholeRoundIsInFlight:
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
			n.enterCall()
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

func (n *peerNetwork) enterCall() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.callsInFlight++
	if n.callsOfTheRound != 0 && n.callsInFlight == n.callsOfTheRound {
		close(n.wholeRoundIsInFlight)
	}
}

func (n *peerNetwork) leaveCall() {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	n.callsInFlight--
}

func asksOfPeersAt(addresses ...string) []peerasks.MatchedItemsAsk {
	asks := make([]peerasks.MatchedItemsAsk, 0, len(addresses))
	for _, address := range addresses {
		asks = append(asks, peerasks.MatchedItemsAsk{Peer: peerAt(address)})
	}

	return asks
}

func TestEveryAskThatCameBackCarriesTheAskItAnswers(t *testing.T) {
	t.Parallel()

	network := &peerNetwork{}
	first := network.peerHolding(t, "https://a.example/")
	second := network.peerHolding(t, "https://b.example/")

	answeredAsks := wireTo(&recordedOutcome{}).AskForMatchedItems(
		t.Context(),
		asksOfPeersAt(first, second),
	)

	if len(answeredAsks) != 2 {
		t.Fatalf(
			"AskForMatchedItems collected %d answers, want one for each ask",
			len(answeredAsks),
		)
	}
	heldByAddress := map[string]string{
		first:  "https://a.example/",
		second: "https://b.example/",
	}
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.Items) != 1 ||
			answeredAsk.Items[0].Address != heldByAddress[answeredAsk.Ask.Peer.Address] {
			t.Fatalf("answered ask %+v does not carry the ask its items came back for", answeredAsk)
		}
	}
}

func TestAPeerThatOutlastsTheCallBudgetIsNoAnswer(t *testing.T) {
	t.Parallel()

	network := &peerNetwork{}
	slow := network.peerAnsweringAfter(t, 10*shortCallBudget, searchAnswerHolding(t))
	holder := network.peerHolding(t, "https://a.example/")

	startedAt := time.Now()
	answeredAsks := wireTo(&recordedOutcome{}).
		AskForMatchedItems(callWithin(t, shortCallBudget), asksOfPeersAt(slow, holder))

	if len(answeredAsks) != 1 || answeredAsks[0].Items[0].Address != "https://a.example/" {
		t.Fatalf("AskForMatchedItems = %+v, want only the peer inside the budget", answeredAsks)
	}
	if time.Since(startedAt) < shortCallBudget {
		t.Fatalf(
			"AskForMatchedItems returned after %v, want it to wait out the budget",
			time.Since(startedAt),
		)
	}
}

func TestEveryAskOfOneRoundIsPutAtOnce(t *testing.T) {
	t.Parallel()

	const asksOfTheRound = 12

	network := &peerNetwork{}
	network.expectsARoundOf(asksOfTheRound)
	asks := make([]peerasks.MatchedItemsAsk, 0, asksOfTheRound)
	for range asksOfTheRound {
		asks = append(
			asks,
			peerasks.MatchedItemsAsk{
				Peer: peerAt(
					network.peerAnsweringWhenTheWholeRoundIsInFlight(t, searchAnswerHolding(t)),
				),
			},
		)
	}

	answeredAsks := wireTo(&recordedOutcome{}).
		AskForMatchedItems(callWithin(t, peerCallBudget), asks)

	if len(answeredAsks) != asksOfTheRound {
		t.Fatalf(
			"%d asks of the round came back, want all %d: the peers did not answer at once",
			len(answeredAsks),
			asksOfTheRound,
		)
	}
}

func TestAskingNoPeerCollectsNoAnswer(t *testing.T) {
	t.Parallel()

	answeredAsks := wireTo(&recordedOutcome{}).AskForMatchedItems(t.Context(), nil)

	if len(answeredAsks) != 0 {
		t.Fatalf("AskForMatchedItems = %+v, want no answers", answeredAsks)
	}
}
