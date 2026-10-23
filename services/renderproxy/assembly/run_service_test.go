package renderproxyassembly_test

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	renderproxy "github.com/nikitakarpei/yacy-rwi-node/renderproxy/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
)

func TestRunServiceServesTheProxyUntilContextCanceled(t *testing.T) {
	listeners := renderproxy.Listeners{Proxy: listenerFor(t)}
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

	if status := statusAt(
		t,
		"http://"+listeners.Proxy.Addr().String()+"/",
	); status != http.StatusBadRequest {
		t.Fatalf("proxy answered %d to a relative URL, want %d", status, http.StatusBadRequest)
	}
	if !hasMetricFamily(t, registry, "renderproxy_info") {
		t.Error("registry lacks renderproxy_info")
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

func listenerFor(t *testing.T) net.Listener {
	t.Helper()

	listener, err := servergroup.Listen(t.Context(), "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	return listener
}

func TestListenersForListensOnTheProxyAddress(t *testing.T) {
	listeners, err := renderproxy.ListenersFor(
		t.Context(), renderproxy.ServiceConfig{ListenAddr: "127.0.0.1:0"},
	)
	if err != nil {
		t.Fatalf("ListenersFor: %v", err)
	}
	t.Cleanup(func() { _ = listeners.Proxy.Close() })
	if listeners.Proxy.Addr().String() == "127.0.0.1:0" {
		t.Errorf("proxy listener = %s, want a bound port", listeners.Proxy.Addr())
	}
}

func TestListenersForFailsOnAnUnusableAddress(t *testing.T) {
	if _, err := renderproxy.ListenersFor(
		t.Context(), renderproxy.ServiceConfig{ListenAddr: "not-an-address"},
	); err == nil {
		t.Fatal("expected error for an unusable proxy address")
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

func statusAt(t *testing.T, pageURL string) int {
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
	_ = answer.Body.Close()
	return answer.StatusCode
}
