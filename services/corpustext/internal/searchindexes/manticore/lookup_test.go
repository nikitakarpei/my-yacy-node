package manticore_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/manticore"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/textsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

const answeredSearch = `{"hits":{"hits":[{
	"_source":{"title":"Hi","url":"https://example.com/"},
	"highlight":{"content":["words here"]}
}]}}`

type recordedSearch struct {
	path string
	body map[string]any
}

func searchRecorder(t *testing.T, recorded *recordedSearch, answer string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorded.path = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&recorded.body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestManticoreLookupYieldsTheMatchedPages(t *testing.T) {
	var recorded recordedSearch
	server := searchRecorder(t, &recorded, answeredSearch)
	lookup := manticore.NewLookup(server.URL, tablesFor(t), server.Client())

	matches, err := lookup.MatchesFor(context.Background(), textsearch.Search{
		Query: "words", Offset: 20, Limit: 10,
	})
	if err != nil {
		t.Fatalf("matches: %v", err)
	}

	want := []searchdocument.PageMatch{
		{URL: "https://example.com/", Title: "Hi", Snippet: "words here"},
	}
	if !slices.Equal(matches, want) {
		t.Errorf("matches = %+v, want %+v", matches, want)
	}
	if recorded.path != "/search" || recorded.body["table"] != "yacy_text_v1" {
		t.Errorf("path = %q, table = %v", recorded.path, recorded.body["table"])
	}
	if recorded.body["offset"] != 20.0 || recorded.body["limit"] != 10.0 {
		t.Errorf("page = offset %v limit %v", recorded.body["offset"], recorded.body["limit"])
	}
}

func TestManticoreLookupMatchesEveryWordAcrossTheWeightedFields(t *testing.T) {
	var recorded recordedSearch
	server := searchRecorder(t, &recorded, answeredSearch)
	lookup := manticore.NewLookup(server.URL, tablesFor(t), server.Client())

	if _, err := lookup.MatchesFor(context.Background(), textsearch.Search{
		Query: "words", Limit: 10,
	}); err != nil {
		t.Fatalf("matches: %v", err)
	}

	match := recorded.body["query"].(map[string]any)["match"].(map[string]any)
	everyField := match["title,content,auxiliary_text"].(map[string]any)
	if everyField["query"] != "words" || everyField["operator"] != "and" {
		t.Errorf("match = %v", match)
	}
	weights := recorded.body["options"].(map[string]any)["field_weights"]
	encodedWeights, _ := json.Marshal(weights)
	if string(encodedWeights) != `{"auxiliary_text":1,"content":2,"title":6}` {
		t.Errorf("field weights = %s", encodedWeights)
	}
}

func TestManticoreLookupKeepsToTheSearchLanguage(t *testing.T) {
	var recorded recordedSearch
	server := searchRecorder(t, &recorded, answeredSearch)
	lookup := manticore.NewLookup(server.URL, tablesFor(t), server.Client())

	if _, err := lookup.MatchesFor(context.Background(), textsearch.Search{
		Query: "words", Language: "en", Limit: 10,
	}); err != nil {
		t.Fatalf("matches: %v", err)
	}

	filter := recorded.body["query"].(map[string]any)["bool"].(map[string]any)["must"].([]any)[1]
	want := map[string]any{"equals": map[string]any{"language": "en"}}
	encodedFilter, _ := json.Marshal(filter)
	encodedWant, _ := json.Marshal(want)
	if string(encodedFilter) != string(encodedWant) {
		t.Errorf("filter = %s, want %s", encodedFilter, encodedWant)
	}
}
