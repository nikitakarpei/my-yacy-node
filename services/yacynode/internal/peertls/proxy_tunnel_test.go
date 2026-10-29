package peertls_test

import (
	"encoding/base64"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/peerkeytest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/peertls"
)

func TestMain(m *testing.M) {
	os.Exit(runTrustingTheTestServerCertificate(m))
}

func runTrustingTheTestServerCertificate(m *testing.M) int {
	server := httptest.NewTLSServer(http.NotFoundHandler())
	certificate := pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw},
	)
	server.Close()
	directory, err := os.MkdirTemp("", "peertls")
	if err != nil {
		panic(err)
	}
	defer func() { _ = os.RemoveAll(directory) }()
	certificateFile := filepath.Join(directory, "test-server.pem")
	if err := os.WriteFile(certificateFile, certificate, 0o600); err != nil {
		panic(err)
	}
	if err := os.Setenv("SSL_CERT_FILE", certificateFile); err != nil {
		panic(err)
	}

	return m.Run()
}

func TestTransportForAPeerTunnelsThroughAnHTTPProxyWithItsCredentials(t *testing.T) {
	peer := peerServer(t, serverConfigOf(t, peerkeytest.Key(0)))
	authorizations := make(chan string, 1)
	proxy := httptest.NewServer(tunnelHandler(authorizations))
	t.Cleanup(proxy.Close)
	proxyURL := proxyURLOf(t, proxy)
	proxyURL.User = url.UserPassword("operator", "secret")

	err := callOf(
		t,
		peer,
		peertls.NewTransports(proxyURL, networkName),
		peerkeytest.Key(0).Identity(),
	)
	if err != nil {
		t.Fatalf("call through the proxy = %v, want an answer", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("operator:secret"))
	if got := <-authorizations; got != want {
		t.Errorf("Proxy-Authorization = %q, want %q", got, want)
	}
}

func TestTransportForAPeerTunnelsThroughAnHTTPSProxyThatItTrusts(t *testing.T) {
	peer := peerServer(t, serverConfigOf(t, peerkeytest.Key(0)))
	proxy := httptest.NewTLSServer(tunnelHandler(make(chan string, 1)))
	t.Cleanup(proxy.Close)

	err := callOf(
		t,
		peer,
		peertls.NewTransports(proxyURLOf(t, proxy), networkName),
		peerkeytest.Key(0).Identity(),
	)
	if err != nil {
		t.Fatalf("call through the HTTPS proxy = %v, want an answer", err)
	}
}

func TestTransportForAPeerFailsWhenTheProxyRefusesTheTunnel(t *testing.T) {
	peer := peerServer(t, serverConfigOf(t, peerkeytest.Key(0)))
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	t.Cleanup(proxy.Close)

	err := callOf(
		t,
		peer,
		peertls.NewTransports(proxyURLOf(t, proxy), networkName),
		peerkeytest.Key(0).Identity(),
	)
	if err == nil {
		t.Fatal("call answered, want the refused tunnel to fail it")
	}
}

func proxyURLOf(t *testing.T, proxy *httptest.Server) *url.URL {
	t.Helper()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}

	return proxyURL
}

func tunnelHandler(authorizations chan<- string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizations <- r.Header.Get("Proxy-Authorization")
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
