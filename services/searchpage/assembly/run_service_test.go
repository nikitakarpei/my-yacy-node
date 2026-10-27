package searchpageassembly_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	searchpage "github.com/nikitakarpei/yacy-rwi-node/searchpage/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

func TestRunServiceServesTheResultsOfBothSourcesAndRelaysVisits(t *testing.T) {
	corpusText := serverAnswering(t, `{"matches":[{"url":"https://crawled.example/",`+
		`"title":"Crawled page","snippet":"crawled"}]}`)
	yacySearch := serverAnswering(t, `{"channels":[{"items":[{"link":"https://network.example/",`+
		`"title":"Network page","description":"network"}]}]}`)
	visitCrawl := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://crawled.example/", http.StatusFound)
	}))
	t.Cleanup(visitCrawl.Close)
	registry := prometheus.NewRegistry()

	pageURL := runService(t, searchpage.ServiceConfig{
		CorpusTextURL: urlOf(t, corpusText.URL),
		YaCySearchURL: urlOf(t, yacySearch.URL),
		VisitCrawl: &searchpage.VisitCrawlConfig{
			URL:          urlOf(t, visitCrawl.URL),
			LinkSecret:   "secret",
			LinkLifetime: time.Hour,
		},
		YaCySearchCallDeadline: 5 * time.Second,
	}, registry)

	page := bodyAt(t, pageURL+"/search?q=page")
	for _, text := range []string{"Crawled page", "Network page", `href="/visit?`} {
		if !strings.Contains(page, text) {
			t.Errorf("results page lacks %q:\n%s", text, page)
		}
	}
	if destination := redirectOf(
		t,
		pageURL+"/visit?url=https%3A%2F%2Fcrawled.example%2F",
	); destination != "https://crawled.example/" {
		t.Errorf("visit redirected to %q, want the crawled page", destination)
	}
	if !hasMetricFamily(t, registry, "searchpage_info") {
		t.Error("registry lacks searchpage_info")
	}
}

func TestRunServiceWithOnlyTheYaCyNetworkLinksResultsStraightToTheirPages(t *testing.T) {
	yacySearch := serverAnswering(t, `{"channels":[{"items":[{"link":"https://network.example/",`+
		`"title":"Network page","description":"network"}]}]}`)

	pageURL := runService(t, searchpage.ServiceConfig{
		YaCySearchURL:          urlOf(t, yacySearch.URL),
		YaCySearchCallDeadline: 5 * time.Second,
	}, prometheus.NewRegistry())

	page := bodyAt(t, pageURL+"/search?q=page")
	for _, text := range []string{`href="https://network.example/"`, "<dt>!ya</dt>"} {
		if !strings.Contains(page, text) {
			t.Errorf("results page lacks %q:\n%s", text, page)
		}
	}
	if strings.Contains(page, "<dt>!ct</dt>") {
		t.Errorf("results page offers the crawled pages:\n%s", page)
	}
	if status := statusOf(t, pageURL+"/visit?url=https%3A%2F%2Fnetwork.example%2F"); status !=
		http.StatusNotFound {
		t.Errorf("visit answered %d, want no visit route", status)
	}
}

func runService(
	t *testing.T,
	cfg searchpage.ServiceConfig,
	registry *prometheus.Registry,
) string {
	t.Helper()
	listeners := searchpage.Listeners{Page: listenerFor(t)}
	ctx, cancel := context.WithCancel(context.Background())
	serviceDone := make(chan error, 1)
	go func() {
		serviceDone <- searchpage.RunService(ctx, cfg, listeners, registry)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-serviceDone:
			if err != nil {
				t.Errorf("RunService: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("RunService did not stop after context cancel")
		}
	})
	return "http://" + listeners.Page.Addr().String()
}

func TestListenersForListensOnThePageAddress(t *testing.T) {
	listeners, err := searchpage.ListenersFor(
		t.Context(), searchpage.ServiceConfig{ListenAddr: "127.0.0.1:0"},
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
	if _, err := searchpage.ListenersFor(
		t.Context(), searchpage.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable page address")
	}
}

func TestListenersCloseReleasesThePageAddress(t *testing.T) {
	listeners, err := searchpage.ListenersFor(
		t.Context(), searchpage.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}

	if err := listeners.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := listeners.Close(); err == nil {
		t.Error("second Close = nil, want the closed listener refused")
	}
	reopened, err := servergroup.Listen(t.Context(), listeners.Page.Addr().String())
	if err != nil {
		t.Fatalf("listen on the closed page address: %v", err)
	}
	_ = reopened.Close()
}

func serverAnswering(t *testing.T, answer string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, answer)
	}))
	t.Cleanup(server.Close)
	return server
}

func urlOf(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse %q: %v", raw, err)
	}
	return parsed
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
	answer := answerTo(t, http.DefaultClient, pageURL)
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

func statusOf(t *testing.T, target string) int {
	t.Helper()
	answer := answerTo(t, http.DefaultClient, target)
	defer func() { _ = answer.Body.Close() }()
	return answer.StatusCode
}

func redirectOf(t *testing.T, visitURL string) string {
	t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	answer := answerTo(t, client, visitURL)
	defer func() { _ = answer.Body.Close() }()
	return answer.Header.Get("Location")
}

func answerTo(t *testing.T, client *http.Client, target string) *http.Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	answer, err := client.Do(request)
	if err != nil {
		t.Fatalf("GET %s: %v", target, err)
	}
	return answer
}
