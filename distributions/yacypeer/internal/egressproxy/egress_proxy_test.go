package egressproxy_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
)

const stopDeadline = 10 * time.Second

type logBuffer struct {
	buffer bytes.Buffer
	mu     sync.Mutex
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, _ = b.buffer.Write(p)
	return len(p), nil
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func runningProxy(t *testing.T) *egressproxy.EgressProxy {
	t.Helper()
	proxy, err := egressproxy.Listen(t.Context())
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- proxy.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("Run: %v", err)
			}
		case <-time.After(stopDeadline):
			t.Error("Run did not stop after cancel")
		}
	})
	return proxy
}

func clientThrough(t *testing.T, proxy *egressproxy.EgressProxy) *http.Client {
	t.Helper()
	return &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxy.URL())}}
}

func TestURLNamesTheLoopbackInterface(t *testing.T) {
	proxy, err := egressproxy.Listen(t.Context())
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	if got := proxy.URL().String(); !strings.HasPrefix(got, "http://127.0.0.1:") {
		t.Errorf("URL = %q, want a loopback URL", got)
	}
}

func TestRunRefusesARequestToALoopbackAddress(t *testing.T) {
	logs := &logBuffer{}
	previous := slog.Default()
	slog.SetDefault(
		slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	)
	t.Cleanup(func() { slog.SetDefault(previous) })
	reached := make(chan struct{}, 1)
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached <- struct{}{}
	}))
	t.Cleanup(target.Close)
	client := clientThrough(t, runningProxy(t))

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target.URL, nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("request through the proxy: %v", err)
	}
	_ = response.Body.Close()

	if response.StatusCode == http.StatusOK {
		t.Errorf("status = %d, want the request refused", response.StatusCode)
	}
	select {
	case <-reached:
		t.Error("the loopback target received the request")
	default:
	}
	if !strings.Contains(logs.String(), "egress proxy event") {
		t.Errorf("logs = %q, want the proxy decision logged", logs.String())
	}
}

func TestCloseReleasesTheProxyAddress(t *testing.T) {
	proxy, err := egressproxy.Listen(t.Context())
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}

	if err := proxy.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, err := servergroup.Listen(t.Context(), proxy.URL().Host)
	if err != nil {
		t.Fatalf("listen on the closed proxy address: %v", err)
	}
	_ = reopened.Close()
}
