package yacydhtsearchpageassembly_test

import (
	"context"
	"html"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	yacydhtsearchpage "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearchpage/assembly"
)

func TestRunServiceServesThePageOverTheYaCyDHTSearch(t *testing.T) {
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w,
			`{"channels":[{"items":[{"title":"YaCy","link":"https://yacy.net/"}]}]}`)
	}))
	t.Cleanup(search.Close)
	searchURL, err := url.Parse(search.URL)
	if err != nil {
		t.Fatalf("parse %q: %v", search.URL, err)
	}
	listeners := yacydhtsearchpage.Listeners{Page: listenerFor(t)}
	cfg := yacydhtsearchpage.ServiceConfig{YaCyDHTSearchURL: searchURL}
	registry := prometheus.NewRegistry()

	ctx, cancel := context.WithCancel(t.Context())
	serviceDone := make(chan error, 1)
	go func() {
		serviceDone <- yacydhtsearchpage.RunService(ctx, cfg, listeners, registry)
	}()

	page := bodyAt(t, "http://"+listeners.Page.Addr().String()+"/?query=yacy")
	if !strings.Contains(page, `href="https://yacy.net/"`) {
		t.Fatalf("page lacks the result:\n%s", page)
	}
	if !hasMetricFamily(t, registry, "yacydhtsearchpage_info") {
		t.Error("registry lacks yacydhtsearchpage_info")
	}

	cancel()
	select {
	case err := <-serviceDone:
		if err != nil {
			t.Fatalf("RunService: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("RunService did not stop after context cancel")
	}
}

func TestRunServiceRoutesEachResultThroughVisitcrawl(t *testing.T) {
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w,
			`{"channels":[{"items":[{"title":"YaCy","link":"https://yacy.net/"}]}]}`)
	}))
	t.Cleanup(search.Close)
	visits := make(chan url.Values, 1)
	visitcrawl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visits <- r.URL.Query()
		http.Redirect(w, r, "https://yacy.net/", http.StatusFound)
	}))
	t.Cleanup(visitcrawl.Close)
	listeners := yacydhtsearchpage.Listeners{Page: listenerFor(t)}
	cfg := yacydhtsearchpage.ServiceConfig{
		YaCyDHTSearchURL: urlOf(t, search.URL),
		VisitRouting: &yacydhtsearchpage.VisitRouting{
			VisitcrawlURL: urlOf(t, visitcrawl.URL),
			LinkSecret:    "shared-secret",
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	serviceDone := make(chan error, 1)
	go func() {
		serviceDone <- yacydhtsearchpage.RunService(ctx, cfg, listeners, prometheus.NewRegistry())
	}()
	pageOrigin := "http://" + listeners.Page.Addr().String()

	page := bodyAt(t, pageOrigin+"/?query=yacy")
	start := strings.Index(page, `href="/visit?`)
	if start < 0 {
		t.Fatalf("page lacks a visit link:\n%s", page)
	}
	link := html.UnescapeString(
		page[start+len(`href="`) : start+strings.Index(page[start:], `" rel`)],
	)
	if destination := redirectOf(t, pageOrigin+link); destination != "https://yacy.net/" {
		t.Errorf("visit redirected to %q, want the redirect of visitcrawl", destination)
	}
	if visit := <-visits; visit.Get("url") != "https://yacy.net/" {
		t.Errorf("visitcrawl got the visit of %q, want the result", visit.Get("url"))
	}

	cancel()
	if err := <-serviceDone; err != nil {
		t.Fatalf("RunService: %v", err)
	}
}

func urlOf(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return parsed
}

func redirectOf(t *testing.T, visitURL string) string {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, visitURL, nil)
	if err != nil {
		t.Fatalf("build the visit: %v", err)
	}
	answer, err := client.Do(request)
	if err != nil {
		t.Fatalf("visit: %v", err)
	}
	defer func() { _ = answer.Body.Close() }()
	return answer.Header.Get("Location")
}

func TestListenersForListensOnThePageAddress(t *testing.T) {
	listeners, err := yacydhtsearchpage.ListenersFor(
		t.Context(), yacydhtsearchpage.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.Page.Close() })
	if listeners.Page.Addr().String() == "127.0.0.1:0" {
		t.Errorf("page listener = %s, want a bound port", listeners.Page.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	if _, err := yacydhtsearchpage.ListenersFor(
		t.Context(), yacydhtsearchpage.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable page address")
	}
}

func hasMetricFamily(t *testing.T, registry *prometheus.Registry, name string) bool {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, family := range families {
		if family.GetName() == name {
			return true
		}
	}
	return false
}

func listenerFor(t *testing.T) net.Listener {
	t.Helper()
	listener, err := servergroup.Listen(t.Context(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func bodyAt(t *testing.T, pageURL string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	answer, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", pageURL, err)
	}
	defer func() { _ = answer.Body.Close() }()
	body, err := io.ReadAll(answer.Body)
	if err != nil {
		t.Fatalf("read %s: %v", pageURL, err)
	}
	if answer.StatusCode != http.StatusOK {
		t.Fatalf("GET %s answered %d", pageURL, answer.StatusCode)
	}
	return string(body)
}
