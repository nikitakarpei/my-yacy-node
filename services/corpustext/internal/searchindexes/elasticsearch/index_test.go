package elasticsearch_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/elasticsearch"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

func TestElasticsearchIndexPutsDocumentByID(t *testing.T) {
	var gotPath string
	var gotDoc searchdocument.Document
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotDoc); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	index := elasticsearch.New(server.URL, indexesFor(t), server.Client())
	document := searchdocument.Document{
		URL:       "https://example.com/",
		Title:     "Hi",
		Content:   "words here",
		CrawledAt: time.Unix(0, 0).UTC(),
		Language:  "en",
	}
	if err := index.Index(context.Background(), document); err != nil {
		t.Fatalf("index: %v", err)
	}
	if !strings.HasPrefix(gotPath, "/yacy_text_v1_en/_doc/") {
		t.Errorf("path = %q", gotPath)
	}
	if gotDoc.Title != "Hi" || gotDoc.URL != "https://example.com/" ||
		gotDoc.Content != "words here" {
		t.Errorf("document = %+v", gotDoc)
	}
}

func TestElasticsearchIndexIsStableForSameURL(t *testing.T) {
	var paths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	index := elasticsearch.New(server.URL, indexesFor(t), server.Client())
	document := searchdocument.Document{URL: "https://example.com/"}
	for range 2 {
		if err := index.Index(context.Background(), document); err != nil {
			t.Fatalf("index: %v", err)
		}
	}
	if paths[0] != paths[1] {
		t.Errorf("path not stable: %q != %q", paths[0], paths[1])
	}
}

func TestElasticsearchIndexReturnsErrorOnFailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	index := elasticsearch.New(server.URL, indexesFor(t), server.Client())
	err := index.Index(
		context.Background(),
		searchdocument.Document{URL: "https://example.com/"},
	)
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}
