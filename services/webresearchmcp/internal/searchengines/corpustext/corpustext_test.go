package corpustext_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/corpustext"
)

const (
	searchQuery    = "research subject"
	searchDeadline = 5 * time.Second
	answeredJSON   = `{"matches":[
		{"url":"https://example.org/page","title":"Research subject",
		 "snippet":"A page about the research subject."},
		{"url":"https://example.org/other","title":"Other","snippet":"Other page."}
	]}`
)

func corpusTextAnswering(t *testing.T, answer string, statusCode int) (string, *[]*http.Request) {
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

func TestSearchAnswersWithEveryMatchInOrder(t *testing.T) {
	corpusTextURL, _ := corpusTextAnswering(t, answeredJSON, http.StatusOK)
	results, err := corpustext.NewCorpusText(corpusTextURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %v, want the 2 matches", results)
	}
	first := results[0]
	if first.URL != "https://example.org/page" || first.Title != "Research subject" ||
		first.Snippet != "A page about the research subject." {
		t.Errorf("first result = %+v", first)
	}
}

func TestSearchAsksTheSearchPathForTheQuery(t *testing.T) {
	corpusTextURL, asked := corpusTextAnswering(t, answeredJSON, http.StatusOK)
	if _, err := corpustext.NewCorpusText(corpusTextURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(*asked) != 1 {
		t.Fatalf("corpustext was asked %d times, want once", len(*asked))
	}
	request := (*asked)[0]
	if request.URL.Path != searchdocument.SearchPath {
		t.Errorf("path = %q, want %q", request.URL.Path, searchdocument.SearchPath)
	}
	if got := request.URL.Query().Get(searchdocument.SearchQueryParam); got != searchQuery {
		t.Errorf("query = %q, want %q", got, searchQuery)
	}
}

func TestSearchFailsWhenCorpusTextAnswersWithAnError(t *testing.T) {
	corpusTextURL, _ := corpusTextAnswering(t, "away", http.StatusBadGateway)
	if _, err := corpustext.NewCorpusText(corpusTextURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err == nil {
		t.Fatal("expected error when corpustext answers 502")
	}
}

func TestSearchFailsWhenTheAnswerIsNotJSON(t *testing.T) {
	corpusTextURL, _ := corpusTextAnswering(t, "not json", http.StatusOK)
	if _, err := corpustext.NewCorpusText(corpusTextURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err == nil {
		t.Fatal("expected error when the answer is not json")
	}
}

func TestSearchFailsWhenCorpusTextIsUnreachable(t *testing.T) {
	if _, err := corpustext.NewCorpusText("http://127.0.0.1:1", searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err == nil {
		t.Fatal("expected error when corpustext is unreachable")
	}
}

func TestSearchFailsWhenTheAddressIsUnusable(t *testing.T) {
	if _, err := corpustext.NewCorpusText("http://bad host", searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err == nil {
		t.Fatal("expected error for an unusable address")
	}
}
