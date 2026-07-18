//go:build e2e

package e2e

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/dockernetwork"
	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/lightpanda"
)

func TestRenderproxyRendersScriptedPageEndToEnd(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	originURL := startScriptedOrigin(t, ctx, network.Name)
	lightpanda.Start(t, ctx, network.Name)
	renderproxyURL := startRenderproxy(t, ctx, network.Name, lightpanda.NetworkURL(), nil)

	client := forwardProxyClient(t, renderproxyURL)
	resp, err := client.Get(originURL)
	if err != nil {
		t.Fatalf("proxy request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read proxy response body: %v", err)
	}
	if !strings.Contains(string(body), renderedMarker) {
		t.Fatalf("rendered body missing marker: status=%d body=%q", resp.StatusCode, body)
	}
}

func TestRenderproxyReturnsNonHTMLRawBodyEndToEnd(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	originURL := startNonHTMLOrigin(t, ctx, network.Name)
	lightpanda.Start(t, ctx, network.Name)
	renderproxyURL := startRenderproxy(t, ctx, network.Name, lightpanda.NetworkURL(), nil)

	client := forwardProxyClient(t, renderproxyURL)
	resp, err := client.Get(originURL)
	if err != nil {
		t.Fatalf("proxy request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read proxy response body: %v", err)
	}
	if got := resp.Header.Get("Content-Type"); got != nonhtmlContentType {
		t.Fatalf("content type = %q, want %q", got, nonhtmlContentType)
	}
	if string(body) != nonhtmlPayload {
		t.Fatalf("raw body = %q, want %q", body, nonhtmlPayload)
	}
}

func TestRenderproxyTimesOutHangingOriginEndToEnd(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	originURL := startHangingOrigin(t, ctx, network.Name)
	lightpanda.Start(t, ctx, network.Name)
	renderproxyURL := startRenderproxy(
		t,
		ctx,
		network.Name,
		lightpanda.NetworkURL(),
		map[string]string{
			"RENDERPROXY_REQUEST_DEADLINE": "3s",
		},
	)

	client := forwardProxyClient(t, renderproxyURL)

	start := time.Now()
	resp, err := client.Get(originURL)
	if err != nil {
		t.Fatalf("proxy request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusGatewayTimeout)
	}
	if elapsed >= 20*time.Second {
		t.Fatalf("render deadline not enforced: elapsed %s reached the client timeout", elapsed)
	}
}

func TestRenderproxyRefusesConnectEndToEnd(t *testing.T) {
	ctx := context.Background()

	network := dockernetwork.New(t, ctx)

	startScriptedOrigin(t, ctx, network.Name)
	lightpanda.Start(t, ctx, network.Name)
	renderproxyURL := startRenderproxy(t, ctx, network.Name, lightpanda.NetworkURL(), nil)

	status := connectResponseStatus(t, renderproxyURL)
	if status != http.StatusMethodNotAllowed {
		t.Fatalf("connect status = %d, want %d", status, http.StatusMethodNotAllowed)
	}
}

// connectResponseStatus drives a real CONNECT handshake through renderproxy by
// requesting an https:// target through it; net/http issues the CONNECT and hands
// the raw response to OnProxyConnectResponse before deciding whether to tunnel TLS.
func connectResponseStatus(t *testing.T, renderproxyURL string) int {
	t.Helper()
	proxyURL, err := url.Parse(renderproxyURL)
	if err != nil {
		t.Fatalf("parse renderproxy url: %v", err)
	}

	var status int
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		OnProxyConnectResponse: func(_ context.Context, _ *url.URL, _ *http.Request, connectRes *http.Response) error {
			status = connectRes.StatusCode
			return nil
		},
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	// The connect target need not exist; only the intercepted CONNECT status matters.
	resp, err := client.Get("https://example.invalid/")
	if err == nil {
		_ = resp.Body.Close()
	}
	return status
}

func forwardProxyClient(t *testing.T, renderproxyURL string) *http.Client {
	t.Helper()
	proxyURL, err := url.Parse(renderproxyURL)
	if err != nil {
		t.Fatalf("parse renderproxy url: %v", err)
	}
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
	}
}
