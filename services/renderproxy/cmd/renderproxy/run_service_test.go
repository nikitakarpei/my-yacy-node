package main_test

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	renderproxy "github.com/nikitakarpei/yacy-rwi-node/renderproxy/cmd/renderproxy"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

func TestRunServiceServesMetricsUntilContextCanceled(t *testing.T) {
	listeners := renderproxy.Listeners{Proxy: listenerFor(t), Ops: listenerFor(t)}
	cfg := renderproxy.ServiceConfig{
		CDPURL:            "http://127.0.0.1:1",
		EgressProxyURL:    &url.URL{Scheme: "http", Host: "127.0.0.1:1"},
		RenderConcurrency: 1,
		RequestDeadline:   time.Second,
		MaxResponseBytes:  1024,
	}

	ctx, cancel := context.WithCancel(t.Context())
	serviceDone := make(chan error, 1)
	registry := prometheus.NewRegistry()
	go func() {
		serviceDone <- renderproxy.RunService(ctx, cfg, listeners, registry)
	}()

	waitForHealthy(t, "http://"+listeners.Ops.Addr().String()+"/metrics")

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

func listenerFor(t *testing.T) net.Listener {
	t.Helper()

	listener, err := servergroup.Listen(t.Context(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	return listener
}

func waitForHealthy(t *testing.T, url string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build health request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("ops server never answered at %s: %v", url, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ops server at %s answered %d, want %d", url, resp.StatusCode, http.StatusOK)
	}
}
