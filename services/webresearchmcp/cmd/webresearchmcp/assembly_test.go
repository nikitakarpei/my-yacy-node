package main_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	webresearchmcp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/cmd/webresearchmcp"
)

const (
	servingDeadline  = 5 * time.Second
	servingPollPause = 20 * time.Millisecond
	shutdownDeadline = 10 * time.Second
)

func serviceConfig(scrapeRequestNATSURL string) webresearchmcp.ServiceConfig {
	return webresearchmcp.ServiceConfig{
		SearXNGURL:                   "http://127.0.0.1:1",
		SearXNGSearchDeadline:        time.Second,
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
	natsURL := natstestserver.Start(t)
	cfg := serviceConfig(natsURL)
	listeners := webresearchmcp.Listeners{Tools: listenerFor(t), Ops: listenerFor(t)}

	ctx, stopService := context.WithCancel(context.Background())
	defer stopService()
	runDone := make(chan error, 1)
	go func() { runDone <- webresearchmcp.RunService(ctx, cfg, listeners) }()

	waitUntilServed(t, "http://"+listeners.Ops.Addr().String()+"/metrics")
	waitUntilServed(t, "http://"+listeners.Tools.Addr().String())

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
	deadline := time.Now().Add(servingDeadline)
	for time.Now().Before(deadline) {
		if answered(t, address) {
			return
		}
		time.Sleep(servingPollPause)
	}
	t.Fatalf("%s never answered", address)
}

func answered(t *testing.T, address string) bool {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
	if err != nil {
		t.Fatalf("ask %s: %v", address, err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	return true
}

func TestRunServiceFailsWhenTheScrapeRequestNATSIsAway(t *testing.T) {
	cfg := serviceConfig("nats://127.0.0.1:1")
	listeners := webresearchmcp.Listeners{Tools: listenerFor(t), Ops: listenerFor(t)}

	if err := webresearchmcp.RunService(context.Background(), cfg, listeners); err == nil {
		t.Fatal("service ran while the scrape request nats is away, want an error")
	}
}
