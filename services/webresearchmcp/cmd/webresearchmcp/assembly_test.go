package main_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	webresearchmcp "github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/cmd/webresearchmcp"
)

const (
	servingDeadline  = 5 * time.Second
	servingPollPause = 20 * time.Millisecond
	shutdownDeadline = 10 * time.Second
)

func serviceConfig(t *testing.T, scrapeRequestNATSURL string) webresearchmcp.ServiceConfig {
	t.Helper()
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
		ListenAddr:                   freeAddr(t),
		OpsAddr:                      freeAddr(t),
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("take a free address: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release the free address: %v", err)
	}
	return addr
}

func TestRunServiceServesTheToolsAndTheMetricsUntilItIsStopped(t *testing.T) {
	natsURL := natstestserver.Start(t)
	cfg := serviceConfig(t, natsURL)

	ctx, stopService := context.WithCancel(context.Background())
	defer stopService()
	runDone := make(chan error, 1)
	go func() { runDone <- webresearchmcp.RunService(ctx, cfg) }()

	waitUntilServed(t, "http://"+cfg.OpsAddr+"/metrics")
	waitUntilServed(t, "http://"+cfg.ListenAddr)

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

func TestRunServiceFailsWhenTheOpsAddressCannotBind(t *testing.T) {
	natsURL := natstestserver.Start(t)
	cfg := serviceConfig(t, natsURL)
	cfg.OpsAddr = "127.0.0.1:99999"

	if err := webresearchmcp.RunService(context.Background(), cfg); err == nil {
		t.Fatal("service ran while the ops address cannot bind, want an error")
	}
}

func TestRunServiceFailsWhenTheScrapeRequestNATSIsAway(t *testing.T) {
	cfg := serviceConfig(t, "nats://127.0.0.1:1")

	if err := webresearchmcp.RunService(context.Background(), cfg); err == nil {
		t.Fatal("service ran while the scrape request nats is away, want an error")
	}
}
