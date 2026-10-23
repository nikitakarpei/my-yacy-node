package hitsreply_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/hitsreply"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

func replying(t *testing.T, status int, reply string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, reply)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestPageMatchesForTurnsEachHitIntoAPageMatch(t *testing.T) {
	server := replying(t, http.StatusOK, `{"hits":{"hits":[
		{"_source":{"title":"Hi","url":"https://example.com/"},
		 "highlight":{"content":["words here"]}},
		{"_source":{"title":"Plain","url":"https://example.com/plain"}}
	]}}`)

	matches, err := hitsreply.PageMatchesFor(
		context.Background(), server.Client(), server.URL, []byte(`{}`),
	)
	if err != nil {
		t.Fatalf("matches: %v", err)
	}

	want := []searchdocument.PageMatch{
		{URL: "https://example.com/", Title: "Hi", Snippet: "words here"},
		{URL: "https://example.com/plain", Title: "Plain"},
	}
	if !slices.Equal(matches, want) {
		t.Errorf("matches = %+v, want %+v", matches, want)
	}
}

func TestPageMatchesForFailsOnAFailureStatus(t *testing.T) {
	server := replying(t, http.StatusServiceUnavailable, "down")

	if _, err := hitsreply.PageMatchesFor(
		context.Background(), server.Client(), server.URL, []byte(`{}`),
	); err == nil {
		t.Fatal("expected an error")
	}
}

func TestPageMatchesForFailsOnAnUnreadableReply(t *testing.T) {
	server := replying(t, http.StatusOK, "not json")

	if _, err := hitsreply.PageMatchesFor(
		context.Background(), server.Client(), server.URL, []byte(`{}`),
	); err == nil {
		t.Fatal("expected an error")
	}
}
