package egressproxy_test

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servergroup"
	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/egressproxy"
)

const (
	stopDeadline      = 10 * time.Second
	publicDestination = "203.0.113.5"
)

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

func runningProxy(t *testing.T, upstreamProxyURL *url.URL) *egressproxy.EgressProxy {
	t.Helper()
	proxy, err := egressproxy.Listen(t.Context(), upstreamProxyURL)
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

func upstreamProxy(t *testing.T) (*url.URL, <-chan string) {
	t.Helper()
	requests := make(chan string, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Method + " " + r.RequestURI
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)
	upstreamProxyURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse the upstream proxy URL: %v", err)
	}
	return upstreamProxyURL, requests
}

func requestThrough(t *testing.T, proxy *egressproxy.EgressProxy, target string) int {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("build the request: %v", err)
	}
	response, err := clientThrough(t, proxy).Do(request)
	if err != nil {
		t.Fatalf("request through the proxy: %v", err)
	}
	_ = response.Body.Close()
	return response.StatusCode
}

func tunnelThrough(t *testing.T, proxy *egressproxy.EgressProxy, destination string) string {
	t.Helper()
	connection, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", proxy.URL().Host)
	if err != nil {
		t.Fatalf("connect to the proxy: %v", err)
	}
	defer func() { _ = connection.Close() }()
	if _, err := fmt.Fprintf(connection, "CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n",
		destination, destination); err != nil {
		t.Fatalf("ask for a tunnel: %v", err)
	}
	statusLine, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil {
		t.Fatalf("read the tunnel answer: %v", err)
	}
	return statusLine
}

func awaitUpstreamRequest(t *testing.T, requests <-chan string, want string) {
	t.Helper()
	select {
	case got := <-requests:
		if got != want {
			t.Errorf("upstream proxy got %q, want %q", got, want)
		}
	case <-time.After(stopDeadline):
		t.Errorf("upstream proxy never got %q", want)
	}
}

func TestURLNamesTheLoopbackInterface(t *testing.T) {
	proxy, err := egressproxy.Listen(t.Context(), nil)
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
	client := clientThrough(t, runningProxy(t, nil))

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

func TestRunSendsARequestOnThroughTheUpstreamProxy(t *testing.T) {
	upstreamProxyURL, requests := upstreamProxy(t)
	proxy := runningProxy(t, upstreamProxyURL)

	requestThrough(t, proxy, "http://"+publicDestination+"/seed.txt")

	awaitUpstreamRequest(t, requests, "GET http://"+publicDestination+"/seed.txt")
}

func TestRunSendsATunnelOnThroughTheUpstreamProxy(t *testing.T) {
	upstreamProxyURL, requests := upstreamProxy(t)
	proxy := runningProxy(t, upstreamProxyURL)

	tunnelThrough(t, proxy, publicDestination+":443")

	awaitUpstreamRequest(t, requests, "CONNECT "+publicDestination+":443")
}

func TestRunReachesAnUpstreamProxyThatTheURLNamesByHostName(t *testing.T) {
	upstreamProxyURL, requests := upstreamProxy(t)
	upstreamProxyURL.Host = net.JoinHostPort("localhost", upstreamProxyURL.Port())
	proxy := runningProxy(t, upstreamProxyURL)

	requestThrough(t, proxy, "http://"+publicDestination+"/seed.txt")

	awaitUpstreamRequest(t, requests, "GET http://"+publicDestination+"/seed.txt")
}

func TestRunRefusesARequestToAPrivateAddressBeforeTheUpstreamProxy(t *testing.T) {
	upstreamProxyURL, requests := upstreamProxy(t)
	proxy := runningProxy(t, upstreamProxyURL)

	status := requestThrough(t, proxy, "http://10.0.0.1/seed.txt")
	statusLine := tunnelThrough(t, proxy, "10.0.0.1:443")

	if status == http.StatusBadGateway || strings.Contains(statusLine, "502") {
		t.Errorf("status %d and tunnel %q, want both refused by the proxy", status, statusLine)
	}
	select {
	case got := <-requests:
		t.Errorf("upstream proxy got %q, want nothing", got)
	default:
	}
}

func TestCloseReleasesTheProxyAddress(t *testing.T) {
	proxy, err := egressproxy.Listen(t.Context(), nil)
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
