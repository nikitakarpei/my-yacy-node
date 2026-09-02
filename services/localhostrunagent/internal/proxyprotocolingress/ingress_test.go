package proxyprotocolingress_test

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/pires/go-proxyproto"

	"github.com/nikitakarpei/yacy-rwi-node/localhostrunagent/internal/proxyprotocolingress"
)

const (
	clientAddress        = "203.0.113.9"
	proxyProtocolVersion = 2
)

func TestIngressGivesTheNodeTheAddressOfTheProxyHeader(t *testing.T) {
	received := headersOfProxiedRequest(t,
		"GET /Network.xml HTTP/1.1\r\nHost: name.localhost.run\r\n\r\n",
	)

	if forwarded := received.Get("X-Forwarded-For"); forwarded != clientAddress {
		t.Errorf("X-Forwarded-For = %q, want %s", forwarded, clientAddress)
	}
}

func TestIngressRemovesTheForwardingHeadersOfTheClient(t *testing.T) {
	received := headersOfProxiedRequest(t,
		"GET / HTTP/1.1\r\nHost: name.localhost.run\r\n"+
			"X-Forwarded-For: 192.0.2.7\r\n"+
			"X-Forwarded-Proto: https\r\n"+
			"X-Real-Ip: 192.0.2.7\r\n"+
			"Forwarded: for=192.0.2.7\r\n\r\n",
	)

	if forwarded := received.Get("X-Forwarded-For"); forwarded != clientAddress {
		t.Errorf("X-Forwarded-For = %q, want %s", forwarded, clientAddress)
	}
	for _, name := range []string{"X-Forwarded-Proto", "X-Real-Ip", "Forwarded"} {
		if value := received.Get(name); value != "" {
			t.Errorf("%s = %q, want no value", name, value)
		}
	}
}

func TestIngressKeepsThePublicHostOfTheRequest(t *testing.T) {
	origin, hosts := hostRecordingOrigin(t)
	ingress := listening(t, origin)

	sendProxied(t, ingress.Port(), "GET / HTTP/1.1\r\nHost: name.localhost.run\r\n\r\n")

	if host := <-hosts; host != "name.localhost.run" {
		t.Errorf("Host = %q, want name.localhost.run", host)
	}
}

func TestIngressRefusesAConnectionWithoutAProxyHeader(t *testing.T) {
	origin, requests := headerRecordingOrigin(t)
	ingress := listening(t, origin)

	send(t, ingress.Port(), nil, "GET / HTTP/1.1\r\nHost: name.localhost.run\r\n\r\n")

	select {
	case received := <-requests:
		t.Errorf("the node received a request without a proxy header: %v", received)
	case <-time.After(2 * time.Second):
	}
}

func headersOfProxiedRequest(t *testing.T, wire string) http.Header {
	t.Helper()

	origin, requests := headerRecordingOrigin(t)
	ingress := listening(t, origin)

	sendProxied(t, ingress.Port(), wire)

	select {
	case received := <-requests:
		return received
	case <-time.After(10 * time.Second):
		t.Fatal("the node received no request")

		return nil
	}
}

func headerRecordingOrigin(t *testing.T) (*url.URL, chan http.Header) {
	t.Helper()

	requests := make(chan http.Header, 1)

	return origin(t, func(writer http.ResponseWriter, request *http.Request) {
		requests <- request.Header.Clone()
		writer.WriteHeader(http.StatusNoContent)
	}), requests
}

func hostRecordingOrigin(t *testing.T) (*url.URL, chan string) {
	t.Helper()

	hosts := make(chan string, 1)

	return origin(t, func(writer http.ResponseWriter, request *http.Request) {
		hosts <- request.Host
		writer.WriteHeader(http.StatusNoContent)
	}), hosts
}

func origin(t *testing.T, handle http.HandlerFunc) *url.URL {
	t.Helper()

	server := httptest.NewServer(handle)
	t.Cleanup(server.Close)

	parsed, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse the origin: %v", err)
	}

	return parsed
}

func listening(t *testing.T, origin *url.URL) *proxyprotocolingress.Ingress {
	t.Helper()

	ingress, err := proxyprotocolingress.Listen(t.Context(), proxyprotocolingress.Configuration{
		Origin:        origin,
		EgressAddress: netip.MustParseAddr("127.0.0.1"),
	})
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() {
		if err := ingress.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	return ingress
}

func sendProxied(t *testing.T, port int, wire string) {
	t.Helper()

	send(t, port, proxyHeaderOf(t, clientAddress), wire)
}

func proxyHeaderOf(t *testing.T, client string) []byte {
	t.Helper()

	header := proxyproto.HeaderProxyFromAddrs(
		proxyProtocolVersion,
		&net.TCPAddr{IP: net.ParseIP(client), Port: 51234},
		&net.TCPAddr{IP: net.ParseIP("198.51.100.2"), Port: 80},
	)

	formatted, err := header.Format()
	if err != nil {
		t.Fatalf("format the proxy header: %v", err)
	}

	return formatted
}

func send(t *testing.T, port int, header []byte, wire string) {
	t.Helper()

	dialer := net.Dialer{Timeout: 10 * time.Second}

	connection, err := dialer.DialContext(t.Context(), "tcp",
		net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("dial the ingress: %v", err)
	}
	t.Cleanup(func() { _ = connection.Close() })

	if _, err := connection.Write(header); err != nil {
		t.Fatalf("write the proxy header: %v", err)
	}

	if _, err := fmt.Fprint(connection, wire); err != nil {
		t.Fatalf("write to the ingress: %v", err)
	}

	_, _ = bufio.NewReader(connection).ReadString('\n')
}
