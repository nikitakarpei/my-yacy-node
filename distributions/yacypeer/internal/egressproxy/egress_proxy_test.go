package egressproxy_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
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
	return runningProxyAcross(t, nil)
}

func runningProxyAcross(t *testing.T, overlays []egressproxy.Overlay) *egressproxy.EgressProxy {
	t.Helper()
	proxy, err := egressproxy.Listen(t.Context(), overlays)
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

type redirectingDialer struct {
	target string
	dialed chan string
}

func (d *redirectingDialer) DialContext(
	ctx context.Context,
	network, address string,
) (net.Conn, error) {
	d.dialed <- address
	conn, err := new(net.Dialer).DialContext(ctx, network, d.target)
	if err != nil {
		return nil, fmt.Errorf("dial the overlay target: %w", err)
	}
	return conn, nil
}

func overlayTarget(t *testing.T) *redirectingDialer {
	t.Helper()
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("overlay"))
	}))
	t.Cleanup(target.Close)
	return &redirectingDialer{
		target: target.Listener.Addr().String(),
		dialed: make(chan string, 1),
	}
}

func statusThrough(t *testing.T, proxy *egressproxy.EgressProxy, address string) int {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
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

func TestRunSendsARequestForAnOverlayAddressThroughTheOverlay(t *testing.T) {
	overlayDialer := overlayTarget(t)
	proxy := runningProxyAcross(t, yggdrasilLikeOverlay(overlayDialer))

	status := statusThrough(t, proxy, "http://[200::1]:8090/")

	if status != http.StatusOK {
		t.Errorf("status = %d, want the overlay to answer", status)
	}
	if dialed := <-overlayDialer.dialed; dialed != "[200::1]:8090" {
		t.Errorf("overlay dialed %q, want [200::1]:8090", dialed)
	}
}

type failingDialer struct{}

func (failingDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("overlay unreachable")
}

func yggdrasilLikeOverlay(dialer egressproxy.Dialer) []egressproxy.Overlay {
	return []egressproxy.Overlay{{Addresses: netip.MustParsePrefix("200::/7"), Dialer: dialer}}
}

func tunnelThrough(
	t *testing.T,
	proxy *egressproxy.EgressProxy,
	destination string,
) (net.Conn, *bufio.Reader, int) {
	t.Helper()
	conn, err := new(net.Dialer).DialContext(t.Context(), "tcp", proxy.URL().Host)
	if err != nil {
		t.Fatalf("dial the proxy: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if _, err := fmt.Fprintf(
		conn,
		"CONNECT %s HTTP/1.1\r\nHost: %s\r\n\r\n",
		destination,
		destination,
	); err != nil {
		t.Fatalf("send CONNECT: %v", err)
	}
	reader := bufio.NewReader(conn)
	response, err := http.ReadResponse(reader, &http.Request{Method: http.MethodConnect})
	if err != nil {
		t.Fatalf("read the CONNECT response: %v", err)
	}
	_ = response.Body.Close()
	return conn, reader, response.StatusCode
}

func TestRunTunnelsAConnectToAnOverlayAddressThroughTheOverlay(t *testing.T) {
	overlayDialer := overlayTarget(t)
	proxy := runningProxyAcross(t, yggdrasilLikeOverlay(overlayDialer))

	conn, reader, status := tunnelThrough(t, proxy, "[200::1]:8090")
	if status != http.StatusOK {
		t.Fatalf("CONNECT status = %d, want the tunnel established", status)
	}
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: [200::1]:8090\r\n\r\n"); err != nil {
		t.Fatalf("send the tunnelled request: %v", err)
	}
	response, err := http.ReadResponse(reader, nil)
	if err != nil {
		t.Fatalf("read the tunnelled response: %v", err)
	}
	body, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()

	if string(body) != "overlay" {
		t.Errorf("tunnelled body = %q, want the overlay answer", body)
	}
}

func TestRunAnswersBadGatewayWhenTheOverlayCannotForward(t *testing.T) {
	proxy := runningProxyAcross(t, yggdrasilLikeOverlay(failingDialer{}))

	if status := statusThrough(t, proxy, "http://[200::1]:8090/"); status != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", status, http.StatusBadGateway)
	}
}

func TestRunAnswersBadGatewayWhenTheOverlayCannotTunnel(t *testing.T) {
	proxy := runningProxyAcross(t, yggdrasilLikeOverlay(failingDialer{}))

	if _, _, status := tunnelThrough(t, proxy, "[200::1]:8090"); status != http.StatusBadGateway {
		t.Errorf("CONNECT status = %d, want %d", status, http.StatusBadGateway)
	}
}
