package peerwire_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peerwire"
)

func exchangeEndpoint(t *testing.T, server *httptest.Server) string {
	t.Helper()

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	return parsed.Host
}

func TestExchangePostsFormAndParsesReply(t *testing.T) {
	var (
		gotPath        string
		gotContentType string
		gotForm        url.Values
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		gotForm = r.PostForm
		_, _ = w.Write([]byte("result=ok\n"))
	}))
	defer server.Close()

	msg, err := peerwire.NewMessageExchange(server.Client()).Exchange(
		context.Background(),
		exchangeEndpoint(t, server),
		"/yacy/transferRWI.html",
		url.Values{"iam": {"peer"}},
	)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if gotPath != "/yacy/transferRWI.html" {
		t.Errorf("path = %q, want /yacy/transferRWI.html", gotPath)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("content type = %q", gotContentType)
	}
	if gotForm.Get("iam") != "peer" {
		t.Errorf("form iam = %q, want peer", gotForm.Get("iam"))
	}
	if msg["result"] != "ok" {
		t.Errorf("result = %q, want ok", msg["result"])
	}
}

func TestExchangeReportsStatusAndReportedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream connect error", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := peerwire.NewMessageExchange(server.Client()).Exchange(
		context.Background(),
		exchangeEndpoint(t, server),
		"/yacy/transferRWI.html",
		url.Values{},
	)
	if err == nil {
		t.Fatal("expected error on non-200")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error = %q, want the status", err)
	}
	if !strings.Contains(err.Error(), "upstream connect error") {
		t.Errorf("error = %q, want the reported body", err)
	}
}

func TestExchangeReportsStatusWithoutReportedBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := peerwire.NewMessageExchange(server.Client()).Exchange(
		context.Background(),
		exchangeEndpoint(t, server),
		"/yacy/transferRWI.html",
		url.Values{},
	)
	if err == nil {
		t.Fatal("expected error on non-200")
	}
	if !strings.Contains(err.Error(), "no reported body") {
		t.Errorf("error = %q, want %q", err, "no reported body")
	}
}

func TestExchangeRejectsEmptyEndpoint(t *testing.T) {
	_, err := peerwire.NewMessageExchange(http.DefaultClient).Exchange(
		context.Background(),
		"  ",
		"/yacy/transferRWI.html",
		url.Values{},
	)
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
}

func TestExchangeReportsUnreachablePeer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := exchangeEndpoint(t, server)
	client := server.Client()
	server.Close()

	_, err := peerwire.NewMessageExchange(client).Exchange(
		context.Background(),
		endpoint,
		"/yacy/transferRWI.html",
		url.Values{},
	)
	if err == nil {
		t.Fatal("expected error for a closed peer")
	}
}
