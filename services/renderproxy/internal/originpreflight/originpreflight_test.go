package originpreflight_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/originpreflight"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagefreshness"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

const responseLimit = 1024

type stubRenderer struct {
	page renderedpage.Page
}

func (s *stubRenderer) Render(
	context.Context,
	renderedpage.Target,
) (renderedpage.Page, error) {
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
	renderer := originpreflight.New(inner, egressURL(t, egress.URL), responseLimit)

	page, err := renderer.Render(t.Context(), renderedpage.Target{URL: "http://origin.test/"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if page.StatusCode != http.StatusMovedPermanently {
		t.Fatalf("status = %d", page.StatusCode)
	}
	if page.Location != "https://example.com/final" {
		t.Fatalf("location = %q", page.Location)
	}
}

func TestRenderDelegatesNonRedirect(t *testing.T) {
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	}))
	defer egress.Close()

	inner := &stubRenderer{
		page: renderedpage.Page{StatusCode: 200, Body: []byte("rendered")},
	}
	renderer := originpreflight.New(inner, egressURL(t, egress.URL), responseLimit)

	page, err := renderer.Render(t.Context(), renderedpage.Target{URL: "http://origin.test/"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(page.Body) != "rendered" {
		t.Fatalf("body = %q", page.Body)
	}
}

func TestRenderFailsWhenEgressUnreachable(t *testing.T) {
	inner := &stubRenderer{}
	renderer := originpreflight.New(inner, egressURL(t, "http://127.0.0.1:1"), responseLimit)

	if _, err := renderer.Render(
		t.Context(),
		renderedpage.Target{URL: "http://origin.test/"},
	); err == nil {
		t.Fatal("expected error when egress proxy unreachable")
	}
}

func TestRenderServesANonHypertextBodyWithoutTheBrowser(t *testing.T) {
	const payload = `{"marker":"raw-body-marker"}`
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))
	defer egress.Close()

	inner := &stubRenderer{page: renderedpage.Page{Body: []byte("rendered")}}
	renderer := originpreflight.New(inner, egressURL(t, egress.URL), responseLimit)

	page, err := renderer.Render(
		t.Context(),
		renderedpage.Target{URL: "http://origin.test/payload.json"},
	)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if string(page.Body) != payload {
		t.Fatalf("body = %q, want %q", page.Body, payload)
	}
	if page.ContentType != "application/json" {
		t.Fatalf("content type = %q", page.ContentType)
	}
}

func TestRenderRefusesANonHypertextBodyBeyondTheLimit(t *testing.T) {
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("a"), responseLimit+1))
	}))
	defer egress.Close()

	renderer := originpreflight.New(&stubRenderer{}, egressURL(t, egress.URL), responseLimit)

	_, err := renderer.Render(
		t.Context(),
		renderedpage.Target{URL: "http://origin.test/payload.json"},
	)
	if !errors.Is(err, renderedpage.ErrTooLarge) {
		t.Fatalf("err = %v, want %v", err, renderedpage.ErrTooLarge)
	}
}

func TestRenderRefusesACompressedBodyThatUnpacksBeyondTheLimit(t *testing.T) {
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(bytes.Repeat([]byte("a"), 200*responseLimit)); err != nil {
		t.Fatalf("compress payload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close compressed payload: %v", err)
	}
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(compressed.Bytes())
	}))
	defer egress.Close()

	renderer := originpreflight.New(&stubRenderer{}, egressURL(t, egress.URL), responseLimit)

	_, err := renderer.Render(
		t.Context(),
		renderedpage.Target{URL: "http://origin.test/payload.json"},
	)
	if !errors.Is(err, renderedpage.ErrTooLarge) {
		t.Fatalf("err = %v, want %v", err, renderedpage.ErrTooLarge)
	}
}

func TestRenderStatesClientConditionsOnTheOriginRequest(t *testing.T) {
	statedConditions := make(chan string, 1)
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		statedConditions <- r.Header.Get("If-None-Match")
		w.WriteHeader(http.StatusNotModified)
	}))
	defer egress.Close()

	renderer := originpreflight.New(&stubRenderer{}, egressURL(t, egress.URL), responseLimit)
	clientRequestHeader := http.Header{}
	clientRequestHeader.Set("If-None-Match", `"v1"`)

	_, err := renderer.Render(t.Context(), renderedpage.Target{
		URL:        "http://origin.test/",
		Conditions: pagefreshness.ConditionsOf(clientRequestHeader),
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := <-statedConditions; got != `"v1"` {
		t.Fatalf("if-none-match at origin = %q", got)
	}
}

func TestRenderRelaysNotModifiedWithoutRendering(t *testing.T) {
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("ETag", `"v1"`)
		w.WriteHeader(http.StatusNotModified)
	}))
	defer egress.Close()

	inner := &stubRenderer{page: renderedpage.Page{Body: []byte("rendered")}}
	renderer := originpreflight.New(inner, egressURL(t, egress.URL), responseLimit)

	page, err := renderer.Render(t.Context(), renderedpage.Target{URL: "http://origin.test/"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if page.StatusCode != http.StatusNotModified {
		t.Fatalf("status = %d, want %d", page.StatusCode, http.StatusNotModified)
	}
	if len(page.Body) != 0 {
		t.Fatalf("body = %q, want none", page.Body)
	}
	if got := reuseTermHeaderOf(page).Get("ETag"); got != `"v1"` {
		t.Fatalf("etag = %q", got)
	}
}

func TestRenderCarriesOriginReuseTermsOntoANonHypertextBody(t *testing.T) {
	egress := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=60")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer egress.Close()

	renderer := originpreflight.New(&stubRenderer{}, egressURL(t, egress.URL), responseLimit)

	page, err := renderer.Render(
		t.Context(),
		renderedpage.Target{URL: "http://origin.test/payload.json"},
	)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := reuseTermHeaderOf(page).Get("Cache-Control"); got != "max-age=60" {
		t.Fatalf("cache-control = %q", got)
	}
}

func reuseTermHeaderOf(page renderedpage.Page) http.Header {
	header := http.Header{}
	page.ReuseTerms.StateOn(header)
	return header
}
