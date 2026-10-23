package http_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	textsearchreceivershttp "github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearchreceivers/http"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

type recordingTextSearch struct {
	matches  []searchdocument.PageMatch
	answered bool
	searches []textsearch.Search
}

func (s *recordingTextSearch) MatchesFor(
	_ context.Context,
	search textsearch.Search,
) ([]searchdocument.PageMatch, bool) {
	s.searches = append(s.searches, search)
	return s.matches, s.answered
}

func responseTo(
	t *testing.T,
	search *recordingTextSearch,
	target string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	response := httptest.NewRecorder()
	textsearchreceivershttp.NewMux(search).ServeHTTP(response, request)
	return response
}

func TestTextSearchReceiverAnswersWithTheMatches(t *testing.T) {
	match := searchdocument.PageMatch{
		URL: "https://example.com/", Title: "Hi", Snippet: "words here",
	}
	search := &recordingTextSearch{matches: []searchdocument.PageMatch{match}, answered: true}

	response := responseTo(t, search, "/search?query=words&language=en-US&offset=10&limit=5")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	var body searchdocument.PageMatches
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !slices.Equal(body.Matches, []searchdocument.PageMatch{match}) {
		t.Errorf("matches = %+v", body.Matches)
	}
	want := textsearch.Search{Query: "words", Language: "en", Offset: 10, Limit: 5}
	if !slices.Equal(search.searches, []textsearch.Search{want}) {
		t.Errorf("searches = %+v, want %+v", search.searches, want)
	}
}

func TestTextSearchReceiverTakesTheDefaultPage(t *testing.T) {
	search := &recordingTextSearch{answered: true}

	response := responseTo(t, search, "/search?query=words")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	want := textsearch.Search{Query: "words", Offset: 0, Limit: textsearch.DefaultMatchLimit}
	if !slices.Equal(search.searches, []textsearch.Search{want}) {
		t.Errorf("searches = %+v, want %+v", search.searches, want)
	}
}

func TestTextSearchReceiverRejectsAnUnanswerableSearch(t *testing.T) {
	for _, target := range []string{
		"/search",
		"/search?query=words&offset=first",
		"/search?query=words&limit=0",
	} {
		t.Run(target, func(t *testing.T) {
			search := &recordingTextSearch{answered: true}

			response := responseTo(t, search, target)

			if response.Code != http.StatusBadRequest {
				t.Errorf("status = %d", response.Code)
			}
			if len(search.searches) != 0 {
				t.Errorf("searches = %+v", search.searches)
			}
		})
	}
}

func TestTextSearchReceiverReportsAnIndexThatDidNotAnswer(t *testing.T) {
	response := responseTo(t, &recordingTextSearch{answered: false}, "/search?query=words")

	if response.Code != http.StatusBadGateway {
		t.Errorf("status = %d", response.Code)
	}
}
