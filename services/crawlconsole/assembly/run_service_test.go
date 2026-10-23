package crawlconsoleassembly_test

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

	crawlconsole "github.com/nikitakarpei/yacy-rwi-node/crawlconsole/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

func TestRunServiceServesTheConsoleOverTheCrawlerAPI(t *testing.T) {
	crawlerAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/crawler-status":
			_, _ = io.WriteString(
				w,
				`{"Paused":false,"PendingPageVisits":3,"AmountOfRunningCrawls":1}`,
			)
		default:
			_, _ = io.WriteString(w, `{"Crawls":[]}`)
		}
	}))
	t.Cleanup(crawlerAPI.Close)
	crawlerAPIURL, err := url.Parse(crawlerAPI.URL)
	if err != nil {
		t.Fatalf("parse %q: %v", crawlerAPI.URL, err)
	}
	listeners := crawlconsole.Listeners{Console: listenerFor(t)}
	cfg := crawlconsole.ServiceConfig{CrawlerAPIURL: crawlerAPIURL}
	registry := prometheus.NewRegistry()

	ctx, cancel := context.WithCancel(t.Context())
	serviceDone := make(chan error, 1)
	go func() {
		serviceDone <- crawlconsole.RunService(ctx, cfg, listeners, registry)
	}()

	page := bodyAt(t, "http://"+listeners.Console.Addr().String()+"/")
	if !strings.Contains(page, "1 running · 3 pages pending") {
		t.Fatalf("console page lacks the crawler status:\n%s", page)
	}
	if !hasMetricFamily(t, registry, "crawlconsole_info") {
		t.Error("registry lacks crawlconsole_info")
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

func TestListenersForListensOnTheConsoleAddress(t *testing.T) {
	listeners, err := crawlconsole.ListenersFor(
		t.Context(), crawlconsole.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.Console.Close() })
	if listeners.Console.Addr().String() == "127.0.0.1:0" {
		t.Errorf("console listener = %s, want a bound port", listeners.Console.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	if _, err := crawlconsole.ListenersFor(
		t.Context(), crawlconsole.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable console address")
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
