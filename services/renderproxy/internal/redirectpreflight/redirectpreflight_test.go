package redirectpreflight

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

type stubRenderer struct {
	called bool
	page   renderedpage.Page
}

func (s *stubRenderer) Render(context.Context, string) (renderedpage.Page, error) {
	s.called = true
	return s.page, nil
}

func egressURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse egress url: %v", err)
	}
	return parsed
}

func TestRenderRelaysRedirectVerbatim(t *testing.T) {
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "https://example.com/final")
		w.WriteHeader(http.StatusMovedPermanently)
	}))
	defer egress.Close()

	inner := &stubRenderer{}
	renderer := New(inner, egressURL(t, egress.URL))

	page, err := renderer.Render(t.Context(), "http://origin.test/")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if page.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("status = %d", page.StatusCode)
	}
	if page.Location != "https://example.com/final" {
		t.Fatalf("location = %q", page.Location)
	}
	if inner.called {
		t.Fatal("browser render should not be called on redirect relay")
	}
}

func TestRenderDelegatesNonRedirect(t *testing.T) {
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer egress.Close()

	inner := &stubRenderer{page: renderedpage.Page{StatusCode: 200, Body: []byte("rendered")}}
	renderer := New(inner, egressURL(t, egress.URL))

	page, err := renderer.Render(t.Context(), "http://origin.test/")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !inner.called {
		t.Fatal("browser render should be called for non-redirect")
	}
	if string(page.Body) != "rendered" {
		t.Fatalf("body = %q", page.Body)
	}
}

func TestRenderFailsWhenEgressUnreachable(t *testing.T) {
	inner := &stubRenderer{}
	renderer := New(inner, egressURL(t, "http://127.0.0.1:1"))

	if _, err := renderer.Render(t.Context(), "http://origin.test/"); err == nil {
		t.Fatal("expected error when egress proxy unreachable")
	}
	if inner.called {
		t.Fatal("browser render should not be called when egress fails")
	}
}
