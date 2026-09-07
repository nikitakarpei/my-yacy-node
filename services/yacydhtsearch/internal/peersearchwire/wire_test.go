package peersearchwire_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearchwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

const responseLimit = 1 << 20

type recordedOutcome struct {
	answered      int
	answeredItems []searchresult.Item
	refused       int
	unreachable   int
	unreadable    int
	spent         time.Duration
}

func (r *recordedOutcome) PeerAnswered(
	_ context.Context,
	_ string,
	answeredItems []searchresult.Item,
	spent time.Duration,
) {
	r.answered++
	r.answeredItems = answeredItems
	r.spent = spent
}

func (r *recordedOutcome) PeerRefused(_ context.Context, _ string, _ int, spent time.Duration) {
	r.refused++
	r.spent = spent
}

func (r *recordedOutcome) PeerUnreachable(
	_ context.Context,
	_ string,
	_ error,
	spent time.Duration,
) {
	r.unreachable++
	r.spent = spent
}

func (r *recordedOutcome) PeerAnswerUnreadable(
	_ context.Context,
	_ string,
	_ error,
	spent time.Duration,
) {
	r.unreadable++
	r.spent = spent
}

func peerAnswering(t *testing.T, body string, status int) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(status)
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func searchAnswerHolding(addresses ...string) string {
	resources := make([]yacyproto.SearchResource, 0, len(addresses))
	for _, address := range addresses {
		resources = append(resources, yacyproto.SearchResource{
			Metadata: yacymodel.URLMetadata{Address: address, Title: "Weather"},
		})
	}

	return yacyproto.SearchResponse{
		Count:     len(resources),
		Resources: resources,
	}.Encode().Encode()
}

func TestAPeerAnswerBecomesResultItems(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	wire := peersearchwire.New(
		http.DefaultClient,
		responseLimit,
		peersearchwire.PeerSearchObservers{observer},
	)

	items, replied := wire.Search(
		t.Context(),
		peerAnswering(t, searchAnswerHolding("https://example.org/weather"), http.StatusOK),
		yacyproto.SearchRequest{NetworkName: "freeworld"},
	)

	if !replied || len(items) != 1 || items[0].Address != "https://example.org/weather" {
		t.Fatalf("Search = %+v, %v, want the address the peer reported", items, replied)
	}
	if observer.answered != 1 {
		t.Fatalf("PeerAnswered reported %d times, want once", observer.answered)
	}
	if len(observer.answeredItems) != 1 ||
		observer.answeredItems[0].Address != "https://example.org/weather" {
		t.Fatalf(
			"PeerAnswered reported %+v, want the item the peer carried",
			observer.answeredItems,
		)
	}
	if observer.spent <= 0 {
		t.Fatalf("PeerAnswered reported %v spent, want the time the call took", observer.spent)
	}
}

func TestAPeerThatRefusesTheSearchYieldsNoItems(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	wire := peersearchwire.New(http.DefaultClient, responseLimit, observer)

	items, replied := wire.Search(
		t.Context(),
		peerAnswering(t, "", http.StatusServiceUnavailable),
		yacyproto.SearchRequest{},
	)

	if replied || len(items) != 0 || observer.refused != 1 {
		t.Fatalf("Search = %+v with %d refusals, want none and one", items, observer.refused)
	}
}

func TestAPeerThatHoldsNothingStillReplies(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	wire := peersearchwire.New(http.DefaultClient, responseLimit, observer)

	items, replied := wire.Search(
		t.Context(),
		peerAnswering(t, searchAnswerHolding(), http.StatusOK),
		yacyproto.SearchRequest{},
	)

	if !replied || len(items) != 0 {
		t.Fatalf("Search = %+v, %v, want a reply that carries nothing", items, replied)
	}
}

func TestAPeerThatCannotBeReachedYieldsNoItems(t *testing.T) {
	t.Parallel()

	observer := &recordedOutcome{}
	wire := peersearchwire.New(http.DefaultClient, responseLimit, observer)

	items, replied := wire.Search(t.Context(), "http://127.0.0.1:1", yacyproto.SearchRequest{})

	if replied || len(items) != 0 || observer.unreachable != 1 {
		t.Fatalf("Search = %+v with %d unreachable, want none and one", items, observer.unreachable)
	}
}
