package corpustext_test

import (
	"context"
	"io"
	"net"
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

func TestTheRankingHoldsTheMatchesOfTheSearch(t *testing.T) {
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
	source := corpustext.New(
		urlOf(t, server.URL),
		(&net.Dialer{}).DialContext,
		callDeadline,
		observer,
	)

	ranking, answered := source.RankingFor(
		t.Context(),
		searchqueries.SearchFrom("example :de", "3", searchqueries.Sources()),
	)

	if !answered {
		t.Fatalf("answered = false, failures %v", observer.failures)
	}
	want := searchresults.SourceRanking{
		{Title: "Example", Address: "https://example.org/", Snippet: "An example page"},
	}
	if len(ranking) != 1 || ranking[0] != want[0] {
		t.Errorf("ranking = %+v, want %+v", ranking, want)
	}
	for name, value := range map[string]string{
		"query": "example", "language": "de", "offset": "", "limit": "30",
	} {
		if asked.Get(name) != value {
			t.Errorf("%s = %q, want %q", name, asked.Get(name), value)
		}
	}
}

func TestASearchThatCorpusTextRefusesGivesNoRanking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "the search index did not answer", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	assertNoRanking(t, server.URL)
}

func TestAnAnswerThatIsNotJSONGivesNoRanking(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "<html>")
	}))
	t.Cleanup(server.Close)

	assertNoRanking(t, server.URL)
}

func TestASearchThatCannotReachCorpusTextGivesNoRanking(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()

	assertNoRanking(t, server.URL)
}

func assertNoRanking(t *testing.T, serverURL string) {
	t.Helper()
	observer := &recordingObserver{}
	source := corpustext.New(
		urlOf(t, serverURL),
		(&net.Dialer{}).DialContext,
		callDeadline,
		observer,
	)

	ranking, answered := source.RankingFor(
		t.Context(),
		searchqueries.SearchFrom("example", "", searchqueries.Sources()),
	)

	if answered || ranking != nil {
		t.Errorf("ranking = %v, answered = %t, want no ranking", ranking, answered)
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
