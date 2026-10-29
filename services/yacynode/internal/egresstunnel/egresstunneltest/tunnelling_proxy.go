// Package egresstunneltest runs a CONNECT proxy for tests that call through the egress proxy.
package egresstunneltest

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func StartedProxy(t *testing.T, authorizations chan<- string) *url.URL {
	t.Helper()

	proxy := httptest.NewServer(TunnelHandler(authorizations))
	t.Cleanup(proxy.Close)

	return ProxyURLOf(t, proxy)
}

func ProxyURLOf(t *testing.T, proxy *httptest.Server) *url.URL {
	t.Helper()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}

	return proxyURL
}

func TunnelHandler(authorizations chan<- string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case authorizations <- r.Header.Get("Proxy-Authorization"):
		default:
		}
		var dialer net.Dialer
		upstream, err := dialer.DialContext(r.Context(), "tcp", r.Host)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)

			return
		}
		client, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			_ = upstream.Close()

			return
		}
		_, _ = io.WriteString(client, "HTTP/1.1 200 Connection established\r\n\r\n")
		go func() {
			_, _ = io.Copy(upstream, client)
			_ = upstream.Close()
		}()
		_, _ = io.Copy(client, upstream)
		_ = client.Close()
	})
}
