package visitrelay_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/visitrelay"
)

type recordingObserver struct {
	failures []error
}

func (o *recordingObserver) RelayFailed(_ context.Context, err error) {
	o.failures = append(o.failures, err)
}

func TestTheRelayGivesThePersonTheRedirectOfVisitCrawl(t *testing.T) {
	var visitedQuery string
	visitCrawl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visitedQuery = r.URL.RawQuery
		http.Redirect(w, r, "https://example.org/page", http.StatusFound)
	}))
	t.Cleanup(visitCrawl.Close)
	observer := &recordingObserver{}
	relay := visitrelay.New(urlOf(t, visitCrawl.URL), observer)

	answer := httptest.NewRecorder()
	relay.ServeHTTP(
		answer,
		httptest.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"/visit?url=x&signature=y",
			nil,
		),
	)

	if answer.Code != http.StatusFound {
		t.Errorf("status = %d, want the redirect of visitcrawl", answer.Code)
	}
	if answer.Header().Get("Location") != "https://example.org/page" {
		t.Errorf("location = %q, want the page", answer.Header().Get("Location"))
	}
	if visitedQuery != "url=x&signature=y" {
		t.Errorf("visitcrawl got query %q, want the visit link query", visitedQuery)
	}
	if len(observer.failures) != 0 {
		t.Errorf("failures = %v, want none", observer.failures)
	}
}

func TestAVisitThatCannotReachVisitCrawlIsABadGateway(t *testing.T) {
	visitCrawl := httptest.NewServer(http.NotFoundHandler())
	visitCrawl.Close()
	observer := &recordingObserver{}
	relay := visitrelay.New(urlOf(t, visitCrawl.URL), observer)

	answer := httptest.NewRecorder()
	relay.ServeHTTP(
		answer,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/visit", nil),
	)

	if answer.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want a bad gateway", answer.Code)
	}
	if len(observer.failures) != 1 {
		t.Errorf("failures = %v, want one", observer.failures)
	}
}

func urlOf(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return parsed
}
