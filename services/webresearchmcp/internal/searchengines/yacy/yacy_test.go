package yacy_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/yacy"
)

const (
	searchQuery    = "research subject"
	searchDeadline = 5 * time.Second
	answeredJSON   = `{"channels":[{"title":"YaCy search","items":[
		{"title":"Research subject","link":"https://example.org/page",
		 "description":"A page about the research subject.","pubDate":""},
		{"title":"Other","link":"https://example.org/other","description":"Other page."}
	]}]}`
)

func yacyAnswering(t *testing.T, answer string, statusCode int) (string, *[]*http.Request) {
	t.Helper()
	var asked []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r)
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(server.Close)
	return server.URL, &asked
}

func TestSearchAnswersWithEveryItemInOrder(t *testing.T) {
	yacyURL, _ := yacyAnswering(t, answeredJSON, http.StatusOK)
	results, err := yacy.NewYaCy(yacyURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %v, want the 2 items", results)
	}
	first := results[0]
	if first.URL != "https://example.org/page" || first.Title != "Research subject" ||
		first.Snippet != "A page about the research subject." {
		t.Errorf("first result = %+v", first)
	}
}

func TestSearchAsksTheYaCySearchInterfaceForTheQuery(t *testing.T) {
	yacyURL, asked := yacyAnswering(t, answeredJSON, http.StatusOK)
	if _, err := yacy.NewYaCy(yacyURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(*asked) != 1 {
		t.Fatalf("yacy was asked %d times, want once", len(*asked))
	}
	request := (*asked)[0]
	if request.URL.Path != "/yacysearch.json" {
		t.Errorf("path = %q, want /yacysearch.json", request.URL.Path)
	}
	if got := request.URL.Query().Get("query"); got != searchQuery {
		t.Errorf("query = %q, want %q", got, searchQuery)
	}
}

func TestSearchFailsWhenYaCyAnswersWithAnError(t *testing.T) {
	yacyURL, _ := yacyAnswering(t, "away", http.StatusServiceUnavailable)
	if _, err := yacy.NewYaCy(yacyURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err == nil {
		t.Fatal("expected error when yacy answers 503")
	}
}

func TestSearchFailsWhenTheAnswerIsNotJSON(t *testing.T) {
	yacyURL, _ := yacyAnswering(t, "not json", http.StatusOK)
	if _, err := yacy.NewYaCy(yacyURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err == nil {
		t.Fatal("expected error when the answer is not json")
	}
}

func TestSearchFailsWhenYaCyIsUnreachable(t *testing.T) {
	if _, err := yacy.NewYaCy("http://127.0.0.1:1", searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err == nil {
		t.Fatal("expected error when yacy is unreachable")
	}
}

func TestSearchFailsWhenTheAddressIsUnusable(t *testing.T) {
	if _, err := yacy.NewYaCy("http://bad host", searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err == nil {
		t.Fatal("expected error for an unusable address")
	}
}
