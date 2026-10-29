package egresstunnel_test

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/pem"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/egresstunnel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/egresstunnel/egresstunneltest"
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
	directory, err := os.MkdirTemp("", "egresstunnel")
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

func TestConnectionThroughAnHTTPProxyReachesTheAddressWithTheProxyCredentials(t *testing.T) {
	target := answeringTarget(t)
	authorizations := make(chan string, 1)
	proxyURL := egresstunneltest.StartedProxy(t, authorizations)
	proxyURL.User = url.UserPassword("operator", "secret")

	connection, err := egresstunnel.ConnectionThrough(t.Context(), proxyURL, target)
	if err != nil {
		t.Fatalf("ConnectionThrough = %v, want a tunnel", err)
	}
	assertTargetAnswers(t, connection)
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("operator:secret"))
	if got := <-authorizations; got != want {
		t.Errorf("Proxy-Authorization = %q, want %q", got, want)
	}
}

func TestConnectionThroughAnHTTPSProxyThatItTrustsReachesTheAddress(t *testing.T) {
	target := answeringTarget(t)
	proxy := httptest.NewTLSServer(egresstunneltest.TunnelHandler(nil))
	t.Cleanup(proxy.Close)

	connection, err := egresstunnel.ConnectionThrough(
		t.Context(),
		egresstunneltest.ProxyURLOf(t, proxy),
		target,
	)
	if err != nil {
		t.Fatalf("ConnectionThrough = %v, want a tunnel", err)
	}
	assertTargetAnswers(t, connection)
}

func TestConnectionThroughFailsWhenTheProxyDoesNotOpenAPlainTunnel(t *testing.T) {
	for name, reply := range map[string]string{
		"refused":            "HTTP/1.1 403 Forbidden\r\nContent-Length: 6\r\n\r\ndenied",
		"body announced":     "HTTP/1.1 200 OK\r\nContent-Length: 99999999999\r\n\r\n",
		"chunked body":       "HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n",
		"early tunnel bytes": "HTTP/1.1 200 Connection established\r\n\r\nEARLY",
	} {
		t.Run(name, func(t *testing.T) {
			proxyURL := proxyAnswering(t, reply)

			if _, err := egresstunnel.ConnectionThrough(
				t.Context(),
				proxyURL,
				"192.0.2.1:8443",
			); err == nil {
				t.Fatal("ConnectionThrough opened a tunnel, want it refused")
			}
		})
	}
}

func TestConnectionThroughStopsWhenItsContextEnds(t *testing.T) {
	proxyURL := proxyAnswering(t, "")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := egresstunnel.ConnectionThrough(ctx, proxyURL, "192.0.2.1:8443"); err == nil {
		t.Fatal("ConnectionThrough opened a tunnel after its context ended")
	}
}

func answeringTarget(t *testing.T) string {
	t.Helper()

	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(target.Close)

	return strings.TrimPrefix(target.URL, "http://")
}

func assertTargetAnswers(t *testing.T, connection net.Conn) {
	t.Helper()
	t.Cleanup(func() { _ = connection.Close() })

	if _, err := io.WriteString(
		connection,
		"GET / HTTP/1.1\r\nHost: target\r\nConnection: close\r\n\r\n",
	); err != nil {
		t.Fatalf("write through the tunnel: %v", err)
	}
	status, err := bufio.NewReader(connection).ReadString('\n')
	if err != nil || !strings.HasPrefix(status, "HTTP/1.1 200") {
		t.Fatalf("target answered %q, %v, want 200", status, err)
	}
}

func proxyAnswering(t *testing.T, reply string) *url.URL {
	t.Helper()

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer func() { _ = connection.Close() }()
				if _, err := http.ReadRequest(bufio.NewReader(connection)); err != nil {
					return
				}
				_, _ = io.WriteString(connection, reply)
				_, _ = io.Copy(io.Discard, connection)
			}()
		}
	}()

	return &url.URL{Scheme: "http", Host: listener.Addr().String()}
}
