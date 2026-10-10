package main_test

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

	impersonateproxy "github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/cmd/impersonateproxy"
	"github.com/nikitakarpei/yacy-rwi-node/impersonateproxy/internal/egressproxytest"
)

const pageBody = "<html><title>Page</title></html>"

func TestTheServiceFetchesPagesAsChromeThroughTheEgressProxy(t *testing.T) {
	originHeaders := make(chan http.Header, 1)
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originHeaders <- r.Header
		_, _ = io.WriteString(w, pageBody)
	}))
	defer origin.Close()
	egressProxy := egressproxytest.Start(t)
	cfg := impersonateproxy.ServiceConfig{
		ListenAddr:     reservedAddress(t),
		EgressProxyURL: egressProxy.URL(),
		FetchTimeout:   time.Minute,
		DefaultReferer: "https://www.google.com/",
		OpsAddr:        reservedAddress(t),
	}
	ctx, cancel := context.WithCancel(t.Context())
	serviceDone := make(chan error, 1)
	go func() { serviceDone <- impersonateproxy.RunService(ctx, cfg, prometheus.NewRegistry()) }()

	body := proxiedBodyOf(t, cfg.ListenAddr, origin.URL+"/page")
	cancel()

	headers := <-originHeaders
	if body != pageBody || !strings.Contains(headers.Get("User-Agent"), "Chrome/152.0.0.0") ||
		headers.Get("Referer") != "https://www.google.com/" ||
		len(egressProxy.TunnelTargets()) != 1 {
		t.Fatalf("body %q, origin headers %v, tunnel targets %v",
			body, headers, egressProxy.TunnelTargets())
	}
	if err := <-serviceDone; err != nil {
		t.Fatalf("RunService: %v", err)
	}
}

func reservedAddress(t *testing.T) string {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func proxiedBodyOf(t *testing.T, proxyAddress, address string) string {
	t.Helper()
	proxyURL := &url.URL{Scheme: "http", Host: proxyAddress}
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
	for {
		request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, address, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := client.Do(request)
		if err == nil {
			body, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil {
				t.Fatal(readErr)
			}
			return string(body)
		}
		if t.Context().Err() != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
