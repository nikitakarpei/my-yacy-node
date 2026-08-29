package searxng_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/searchengines/searxng"
)

const (
	searchQuery    = "research subject"
	searchDeadline = 5 * time.Second
	firstResultURL = "https://example.org/page"
	resultTitle    = "Research subject"
	resultSnippet  = "A page about the research subject."
	answeredJSON   = `{"results":[
		{"url":"https://example.org/page","title":"Research subject",
		 "content":"A page about the research subject."},
		{"url":"https://example.org/other","title":"Other","content":"Other page."}
	]}`
)

func searxngAnswering(t *testing.T, answer string, statusCode int) (string, *[]*http.Request) {
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

func TestSearchAnswersWithEveryResultSearXNGReturnsInOrder(t *testing.T) {
	searxngURL, _ := searxngAnswering(t, answeredJSON, http.StatusOK)

	results, err := searxng.NewSearXNG(searxngURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %v, want the 2 searxng returns", results)
	}
	if results[0].URL != firstResultURL {
		t.Errorf("first result url = %q, want %q", results[0].URL, firstResultURL)
	}
	if results[0].Title != resultTitle {
		t.Errorf("first result title = %q, want %q", results[0].Title, resultTitle)
	}
	if results[0].Snippet != resultSnippet {
		t.Errorf("first result snippet = %q, want %q", results[0].Snippet, resultSnippet)
	}
}

func TestSearchAsksForJSONAndForLinksTheResultRouterLeavesAsTheyAre(t *testing.T) {
	searxngURL, asked := searxngAnswering(t, answeredJSON, http.StatusOK)

	if _, err := searxng.NewSearXNG(searxngURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery); err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(*asked) != 1 {
		t.Fatalf("searxng was asked %d times, want once", len(*asked))
	}
	askedFor := (*asked)[0].URL.Query()
	if askedFor.Get("q") != searchQuery {
		t.Errorf("query = %q, want %q", askedFor.Get("q"), searchQuery)
	}
	if askedFor.Get("format") != "json" {
		t.Errorf("format = %q, want json", askedFor.Get("format"))
	}
	if askedFor.Get("disabled_plugins") != "result_link_router" {
		t.Errorf(
			"disabled plugins = %q, want result_link_router",
			askedFor.Get("disabled_plugins"),
		)
	}
}

func TestSearchThatSearXNGRefusesFails(t *testing.T) {
	searxngURL, _ := searxngAnswering(t, "", http.StatusTooManyRequests)

	_, err := searxng.NewSearXNG(searxngURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery)
	if err == nil {
		t.Fatal("search answered without an error, want the refusal of searxng")
	}
}

func TestSearchAnswerThatIsNotJSONFails(t *testing.T) {
	searxngURL, _ := searxngAnswering(t, "not json", http.StatusOK)

	_, err := searxng.NewSearXNG(searxngURL, searchDeadline).
		SearchResultsFor(context.Background(), searchQuery)
	if err == nil {
		t.Fatal("search answered without an error, want the unreadable answer to fail")
	}
}
