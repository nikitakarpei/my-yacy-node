package visitrelay_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/internal/visitrelay"
)

type recordingObserver struct {
	failures []error
}

func (observer *recordingObserver) VisitRelayFailed(_ context.Context, err error) {
	observer.failures = append(observer.failures, err)
}

func visitThrough(relay http.Handler, path string) *httptest.ResponseRecorder {
	answer := httptest.NewRecorder()
	relay.ServeHTTP(answer,
		httptest.NewRequestWithContext(context.Background(), http.MethodGet, path, nil))
	return answer
}

func TestRelayPassesTheVisitToVisitcrawlAndItsRedirectBack(t *testing.T) {
	visits := make(chan *http.Request, 1)
	visitcrawl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visits <- r
		http.Redirect(w, r, "https://yacy.net/", http.StatusFound)
	}))
	t.Cleanup(visitcrawl.Close)
	visitcrawlURL, err := url.Parse(visitcrawl.URL)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	answer := visitThrough(
		visitrelay.New(visitcrawlURL, &recordingObserver{}),
		"/visit?url=https%3A%2F%2Fyacy.net%2F&expires=1&signature=abc",
	)

	if answer.Code != http.StatusFound || answer.Header().Get("Location") != "https://yacy.net/" {
		t.Errorf("answer = %d to %q, want the redirect of visitcrawl", answer.Code,
			answer.Header().Get("Location"))
	}
	visit := <-visits
	if visit.URL.Path != "/visit" || visit.URL.Query().Get("signature") != "abc" {
		t.Errorf("visitcrawl got %s, want the visit with its query", visit.URL)
	}
}

func TestRelayAnswersBadGatewayWhenVisitcrawlDoesNotAnswer(t *testing.T) {
	away, err := url.Parse("http://127.0.0.1:1")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	observer := &recordingObserver{}

	answer := visitThrough(visitrelay.New(away, observer), "/visit?url=x")

	if answer.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", answer.Code)
	}
	if len(observer.failures) != 1 {
		t.Errorf("observed failures = %v, want one", observer.failures)
	}
}
