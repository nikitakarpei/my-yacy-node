package elasticsearch_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/elasticsearch"
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

func TestElasticsearchMatcherYieldsTheMatchedPages(t *testing.T) {
	var recorded recordedSearch
	server := searchRecorder(t, &recorded, answeredSearch)
	matcher := elasticsearch.NewMatcher(server.URL, indexesFor(t), server.Client())

	matches, err := matcher.MatchesFor(context.Background(), textsearch.Search{
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
	if recorded.path != "/yacy_text_v1_*/_search" {
		t.Errorf("path = %q", recorded.path)
	}
	if recorded.body["from"] != 20.0 || recorded.body["size"] != 10.0 {
		t.Errorf("page = from %v size %v", recorded.body["from"], recorded.body["size"])
	}
}

func TestElasticsearchMatcherMatchesEveryWordAcrossTheWeightedFields(t *testing.T) {
	var recorded recordedSearch
	server := searchRecorder(t, &recorded, answeredSearch)
	matcher := elasticsearch.NewMatcher(server.URL, indexesFor(t), server.Client())

	if _, err := matcher.MatchesFor(context.Background(), textsearch.Search{
		Query: "words", Limit: 10,
	}); err != nil {
		t.Fatalf("matches: %v", err)
	}

	query := recorded.body["query"].(map[string]any)["combined_fields"].(map[string]any)
	if query["query"] != "words" || query["operator"] != "and" {
		t.Errorf("query = %v", query)
	}
	fields := query["fields"].([]any)
	if !slices.Equal(fields, []any{"title^6", "content^2", "auxiliary_text^1"}) {
		t.Errorf("fields = %v", fields)
	}
}

func TestElasticsearchMatcherKeepsToTheSearchLanguage(t *testing.T) {
	var recorded recordedSearch
	server := searchRecorder(t, &recorded, answeredSearch)
	matcher := elasticsearch.NewMatcher(server.URL, indexesFor(t), server.Client())

	if _, err := matcher.MatchesFor(context.Background(), textsearch.Search{
		Query: "words", Language: "en", Limit: 10,
	}); err != nil {
		t.Fatalf("matches: %v", err)
	}

	filter := recorded.body["query"].(map[string]any)["bool"].(map[string]any)["filter"]
	want := []any{map[string]any{"term": map[string]any{"language": "en"}}}
	encodedFilter, _ := json.Marshal(filter)
	encodedWant, _ := json.Marshal(want)
	if string(encodedFilter) != string(encodedWant) {
		t.Errorf("filter = %s, want %s", encodedFilter, encodedWant)
	}
}
