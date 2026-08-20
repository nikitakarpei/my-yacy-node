package manticore_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/searchindexes/manticore"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
)

type replaceRequest struct {
	Table    string                  `json:"table"`
	Identity int64                   `json:"id"`
	Document searchdocument.Document `json:"doc"`
}

func TestManticoreIndexReplacesDocumentByIdentity(t *testing.T) {
	var gotPath string
	var got replaceRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	index := manticore.New(server.URL, tablesFor(t), server.Client())
	page := yacycrawlcontract.PageTextRepresentation{
		PageReference: yacycrawlcontract.PageReference{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
			Title:        "Hi",
			CrawledAt:    time.Unix(0, 0).UTC(),
			Language:     "en",
		},
		Text: []byte("words here"),
	}
	if err := index.Index(context.Background(), page); err != nil {
		t.Fatalf("index: %v", err)
	}
	if gotPath != "/replace" {
		t.Errorf("path = %q", gotPath)
	}
	if got.Table != "yacy_text_v1_en" || got.Identity <= 0 {
		t.Errorf("request = %+v", got)
	}
	if got.Document.Title != "Hi" || got.Document.URL != "https://example.com/" ||
		got.Document.Content != "words here" {
		t.Errorf("document = %+v", got.Document)
	}
}

func TestManticoreIndexIsStableForSameURL(t *testing.T) {
	var identities []int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req replaceRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		identities = append(identities, req.Identity)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	index := manticore.New(server.URL, tablesFor(t), server.Client())
	page := yacycrawlcontract.PageTextRepresentation{
		PageReference: yacycrawlcontract.PageReference{
			CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
		},
	}
	for range 2 {
		if err := index.Index(context.Background(), page); err != nil {
			t.Fatalf("index: %v", err)
		}
	}
	if identities[0] != identities[1] {
		t.Errorf("identity not stable: %d != %d", identities[0], identities[1])
	}
}

func TestManticoreIndexReturnsErrorOnFailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	index := manticore.New(server.URL, tablesFor(t), server.Client())
	err := index.Index(
		context.Background(),
		yacycrawlcontract.PageTextRepresentation{
			PageReference: yacycrawlcontract.PageReference{
				CanonicalURL: canonicalurltest.CanonicalURLOf(t, "https://example.com/"),
			},
		},
	)
	if err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}
