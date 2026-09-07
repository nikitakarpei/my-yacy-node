package peersearch_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearchwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const (
	responseLimit = 1 << 20
	callsInFlight = 4
	callBudget    = 4 * time.Second
)

type silentOutcome struct{}

func (silentOutcome) PeerAnswered(
	context.Context, string, []searchresult.Item, time.Duration,
) {
}
func (silentOutcome) PeerRefused(context.Context, string, int, time.Duration)            {}
func (silentOutcome) PeerUnreachable(context.Context, string, error, time.Duration)      {}
func (silentOutcome) PeerAnswerUnreadable(context.Context, string, error, time.Duration) {}

func peerHolding(t *testing.T, addresses ...string) string {
	t.Helper()

	resources := make([]yacyproto.SearchResource, 0, len(addresses))
	for _, address := range addresses {
		resources = append(resources, yacyproto.SearchResource{
			Metadata: yacymodel.URLMetadata{Address: address},
		})
	}
	body := yacyproto.SearchResponse{Count: len(resources), Resources: resources}.Encode().Encode()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func askablePeerAt(t *testing.T, symbol string, address string) peerdirectory.AskablePeer {
	t.Helper()

	return peerdirectory.AskablePeer{Hash: yacymodel.WordHash(symbol), Address: address}
}

func peersCalling(t *testing.T) peersearch.Peers {
	t.Helper()

	return peersearch.New(
		peersearchwire.New(http.DefaultClient, responseLimit, silentOutcome{}),
		callsInFlight,
		callBudget,
	)
}

func TestEveryAskedPeerThatHoldsSomethingAnswers(t *testing.T) {
	t.Parallel()

	answers := peersCalling(t).Ask(t.Context(), []peerdirectory.AskablePeer{
		askablePeerAt(t, "first", peerHolding(t, "https://a.example/")),
		askablePeerAt(t, "second", peerHolding(t, "https://b.example/")),
	}, yacyproto.SearchRequest{})

	if len(answers) != 2 {
		t.Fatalf("Ask collected %d answers, want 2", len(answers))
	}
}

func TestAPeerWithNothingToSayStillAnswers(t *testing.T) {
	t.Parallel()

	answers := peersCalling(t).Ask(t.Context(), []peerdirectory.AskablePeer{
		askablePeerAt(t, "empty", peerHolding(t)),
		askablePeerAt(t, "holder", peerHolding(t, "https://a.example/")),
	}, yacyproto.SearchRequest{})

	if len(answers) != 2 || len(answers[0].Items) != 0 ||
		answers[1].Items[0].Address != "https://a.example/" {
		t.Fatalf("Ask = %+v, want one empty answer and one that holds something", answers)
	}
}

func TestAPeerThatCannotBeReachedIsNoAnswer(t *testing.T) {
	t.Parallel()

	answers := peersCalling(t).Ask(t.Context(), []peerdirectory.AskablePeer{
		askablePeerAt(t, "gone", "http://127.0.0.1:1"),
		askablePeerAt(t, "holder", peerHolding(t, "https://a.example/")),
	}, yacyproto.SearchRequest{})

	if len(answers) != 1 || answers[0].Items[0].Address != "https://a.example/" {
		t.Fatalf("Ask = %+v, want only the peer that replied", answers)
	}
}

func TestAskingNoPeerCollectsNoAnswer(t *testing.T) {
	t.Parallel()

	if answers := peersCalling(
		t,
	).Ask(t.Context(), nil, yacyproto.SearchRequest{}); len(
		answers,
	) != 0 {
		t.Fatalf("Ask = %+v, want no answers", answers)
	}
}
