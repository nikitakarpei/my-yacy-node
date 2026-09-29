package tunnel_test

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/egresstransports/tunnel"
)

type tunnellingProxy struct {
	originAddress string
	tunnelTargets chan string
}

func (p tunnellingProxy) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	p.tunnelTargets <- request.Host
	var dialer net.Dialer
	origin, err := dialer.DialContext(request.Context(), "tcp", p.originAddress)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
	client, buffered, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return
	}
	go func() {
		_, _ = io.Copy(origin, buffered)
		_ = origin.Close()
	}()
	_, _ = io.Copy(client, origin)
	_ = client.Close()
}

func TestAnHTTPSPageGoesThroughATunnelToItsOrigin(t *testing.T) {
	originRequests := make(chan string, 1)
	origin := httptest.NewTLSServer(
		http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			originRequests <- request.Host + request.RequestURI
			_, _ = io.WriteString(w, "page")
		}),
	)
	defer origin.Close()
	proxy := tunnellingProxy{
		originAddress: origin.Listener.Addr().String(),
		tunnelTargets: make(chan string, 1),
	}
	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()
	proxyURL, _ := url.Parse(proxyServer.URL)
	roots := x509.NewCertPool()
	roots.AddCert(origin.Certificate())
	request, _ := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://example.com/page?q=1",
		nil,
	)

	response, err := tunnel.New(proxyURL, &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}).
		RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)

	target, originRequest := <-proxy.tunnelTargets, <-originRequests
	if target != "example.com:443" || originRequest != "example.com/page?q=1" ||
		string(body) != "page" {
		t.Fatalf("tunnel target %s, origin request %s, body %q", target, originRequest, body)
	}
}
