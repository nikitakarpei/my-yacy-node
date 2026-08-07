package elasticsearchindex_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/elasticsearchindex"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
)

func indexesFor(t *testing.T) languageindex.LanguageIndexes {
	t.Helper()
	indexes, err := languageindex.IndexesFor("yacy_text", []string{"en"})
	if err != nil {
		t.Fatalf("indexes: %v", err)
	}
	return indexes
}

func TestElasticsearchSchemaCreatesOneIndexPerLanguage(t *testing.T) {
	created := map[string]map[string]any{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			var definition map[string]any
			if err := json.NewDecoder(r.Body).Decode(&definition); err != nil {
				t.Errorf("decode definition: %v", err)
			}
			created[r.URL.Path] = definition
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = io.WriteString(w, "{}")
	}))
	defer server.Close()

	schema := elasticsearchindex.NewElasticsearchSchema(
		server.URL, indexesFor(t), server.Client(),
	)
	if err := schema.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}

	catchAll, ok := created["/yacy_text_v1_und"]
	if !ok {
		t.Fatalf("created = %v, want the catch-all index", created)
	}
	if analyzerTypeIn(t, catchAll) != "custom" {
		t.Errorf("catch-all analyzer = %v", catchAll)
	}
	english, ok := created["/yacy_text_v1_en"]
	if !ok {
		t.Fatalf("created = %v, want the english index", created)
	}
	if analyzerTypeIn(t, english) != "english" {
		t.Errorf("english analyzer = %v", english)
	}
}

func analyzerTypeIn(t *testing.T, definition map[string]any) string {
	t.Helper()
	settings, ok := definition["settings"].(map[string]any)
	if !ok {
		t.Fatalf("definition = %v, want settings", definition)
	}
	analysis := settings["analysis"].(map[string]any)
	analyzer := analysis["analyzer"].(map[string]any)["corpus_text"].(map[string]any)
	return analyzer["type"].(string)
}

func TestElasticsearchSchemaAcceptsAnExistingIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, `{"error":{"type":"resource_already_exists_exception"}}`)
			return
		}
		_, _ = io.WriteString(w, "{}")
	}))
	defer server.Close()

	schema := elasticsearchindex.NewElasticsearchSchema(
		server.URL, indexesFor(t), server.Client(),
	)
	if err := schema.Bootstrap(context.Background()); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
}

func TestElasticsearchSchemaFailsOnCreateRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	schema := elasticsearchindex.NewElasticsearchSchema(
		server.URL, indexesFor(t), server.Client(),
	)
	if err := schema.Bootstrap(context.Background()); err == nil {
		t.Fatal("expected error when the index cannot be created")
	}
}
