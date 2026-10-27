package corpustext_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/resultsources/corpustext"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchresults"
)

const callDeadline = 5 * time.Second

type recordingObserver struct {
	failures []error
}

func (o *recordingObserver) SearchFailed(_ context.Context, err error) {
	o.failures = append(o.failures, err)
}

func TestThePageHoldsTheMatchesOfTheSearch(t *testing.T) {
	var asked url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			http.NotFound(w, r)
			return
		}
		asked = r.URL.Query()
		_, _ = io.WriteString(w, `{"matches":[{"url":"https://example.org/","title":"Example",`+
			`"snippet":"An example page"}]}`)
	}))
	t.Cleanup(server.Close)
	observer := &recordingObserver{}
	source := corpustext.New(urlOf(t, server.URL), callDeadline, observer)

	page, answered := source.PageFor(
		t.Context(),
		searchqueries.SearchFrom("example :de", "3", searchqueries.Sources()),
	)

	if !answered {
		t.Fatalf("answered = false, failures %v", observer.failures)
	}
	want := searchresults.SourcePage{
		{Title: "Example", Address: "https://example.org/", Snippet: "An example page"},
	}
	if len(page) != 1 || page[0] != want[0] {
		t.Errorf("page = %+v, want %+v", page, want)
	}
	for name, value := range map[string]string{
		"query": "example", "language": "de", "offset": "20", "limit": "10",
	} {
		if asked.Get(name) != value {
			t.Errorf("%s = %q, want %q", name, asked.Get(name), value)
		}
	}
}

func TestASearchThatCorpusTextRefusesGivesNoPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "the search index did not answer", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	assertNoPage(t, server.URL)
}

func TestAnAnswerThatIsNotJSONGivesNoPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<html>")
	}))
	t.Cleanup(server.Close)

	assertNoPage(t, server.URL)
}

func TestASearchThatCannotReachCorpusTextGivesNoPage(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	assertNoPage(t, server.URL)
}

func assertNoPage(t *testing.T, serverURL string) {
	t.Helper()
	observer := &recordingObserver{}
	source := corpustext.New(urlOf(t, serverURL), callDeadline, observer)

	page, answered := source.PageFor(
		t.Context(),
		searchqueries.SearchFrom("example", "", searchqueries.Sources()),
	)

	if answered || page != nil {
		t.Errorf("page = %v, answered = %t, want no page", page, answered)
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
