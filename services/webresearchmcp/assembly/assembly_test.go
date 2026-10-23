package webresearchmcpassembly_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	webresearchmcp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/assembly"
)

const (
	servingDeadline  = 5 * time.Second
	shutdownDeadline = 10 * time.Second
)

func serviceConfig(scrapeRequestNATSURL string) webresearchmcp.ServiceConfig {
	return webresearchmcp.ServiceConfig{
		WebSearchEngine:              webresearchmcp.WebSearchEngineSearXNG,
		WebSearchDeadline:            time.Second,
		SearXNGURL:                   "http://127.0.0.1:1",
		ScrapeRequestNATSURL:         scrapeRequestNATSURL,
		PageFetchWait:                time.Second,
		PageScrapeTolerance:          webresearchmcp.DefaultPageScrapeTolerance,
		CorpusMarkdownAddr:           "127.0.0.1:1",
		CorpusMarkdownRecallDeadline: time.Second,
		PageFetchCharacterLimit:      webresearchmcp.DefaultPageFetchCharacterLimit,
		SearchResultLimit:            webresearchmcp.DefaultSearchResultLimit,
		ToolCallConcurrency:          webresearchmcp.DefaultToolCallConcurrency,
	}
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

func TestRunServiceServesTheToolsAndTheMetricsUntilItIsStopped(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t))
	runUntilStopped(t, cfg)
}

func TestRunServiceServesTheToolsOverTheSearchEnginesOfTheStack(t *testing.T) {
	cfg := serviceConfig(natstestserver.Start(t))
	cfg.WebSearchEngine = webresearchmcp.WebSearchEngineStack
	cfg.YaCyDHTSearchURL = "http://127.0.0.1:1"
	cfg.CorpusTextURL = "http://127.0.0.1:1"
	runUntilStopped(t, cfg)
}

func runUntilStopped(t *testing.T, cfg webresearchmcp.ServiceConfig) {
	t.Helper()
	listeners := webresearchmcp.Listeners{Tools: listenerFor(t)}
	registry := prometheus.NewRegistry()

	ctx, stopService := context.WithCancel(context.Background())
	defer stopService()
	runDone := make(chan error, 1)
	go func() { runDone <- webresearchmcp.RunService(ctx, cfg, listeners, registry) }()

	waitUntilServed(t, "http://"+listeners.Tools.Addr().String())
	if !registryCarries(t, registry, "webresearchmcp_info") {
		t.Error("registry carries no webresearchmcp_info metric")
	}

	stopService()
	select {
	case err := <-runDone:
		if err != nil {
			t.Errorf("run: %v", err)
		}
	case <-time.After(shutdownDeadline):
		t.Fatal("service did not shut down after cancel")
	}
}

func waitUntilServed(t *testing.T, address string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), servingDeadline)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		t.Fatalf("ask %s: %v", address, err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s never answered: %v", address, err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
}

func registryCarries(t *testing.T, registry *prometheus.Registry, metricName string) bool {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, family := range families {
		if family.GetName() == metricName {
			return true
		}
	}
	return false
}

func TestRunServiceFailsWhenTheScrapeRequestNATSIsAway(t *testing.T) {
	cfg := serviceConfig("nats://127.0.0.1:1")
	listeners := webresearchmcp.Listeners{Tools: listenerFor(t)}

	if err := webresearchmcp.RunService(
		context.Background(), cfg, listeners, prometheus.NewRegistry(),
	); err == nil {
		t.Fatal("service ran while the scrape request nats is away, want an error")
	}
}

func TestListenersForListensOnTheToolsAddress(t *testing.T) {
	listeners, err := webresearchmcp.ListenersFor(
		t.Context(), webresearchmcp.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.Tools.Close() })
	if listeners.Tools.Addr().String() == "127.0.0.1:0" {
		t.Errorf("tools listener = %s, want a bound port", listeners.Tools.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	if _, err := webresearchmcp.ListenersFor(
		t.Context(), webresearchmcp.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable tools address")
	}
}
