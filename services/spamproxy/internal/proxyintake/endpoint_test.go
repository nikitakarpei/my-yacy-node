package proxyintake_test

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/proxyintake"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/requestrelay"
)

type scriptedRelay struct {
	addresses []canonicalurl.CanonicalURL
	cutsShort bool
}

func (r *scriptedRelay) ReplyTo(
	_ context.Context,
	_ string,
	address canonicalurl.CanonicalURL,
	_ http.Header,
	reply requestrelay.Reply,
) {
	r.addresses = append(r.addresses, address)
	reply.SendHead(http.StatusOK, http.Header{"Content-Length": {"10"}})
	_, _ = reply.Write([]byte("page"))
	if r.cutsShort {
		reply.CutShort()
	}
}

type refusalRecord struct {
	methods []string
	targets []string
}

func (r *refusalRecord) MethodRefused(_ context.Context, method string) {
	r.methods = append(r.methods, method)
}

func (r *refusalRecord) TargetRefused(_ context.Context, target string, _ error) {
	r.targets = append(r.targets, target)
}

func TestAnAbsoluteWebAddressIsRelayed(t *testing.T) {
	relay := &scriptedRelay{}
	server := httptest.NewServer(proxyintake.New(relay, &refusalRecord{}))
	defer server.Close()

	response := proxiedResponseTo(t, server, "GET http://Site.example/page HTTP/1.1")
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK ||
		relay.addresses[0].String() != "http://site.example/page" {
		t.Fatalf("status %d, addresses %v", response.StatusCode, relay.addresses)
	}
	if _, found := response.Header["Content-Type"]; found {
		t.Fatalf("headers %v", response.Header)
	}
}

func TestOnlyGetAndHeadAreRelayed(t *testing.T) {
	relay := &scriptedRelay{}
	refusals := &refusalRecord{}
	server := httptest.NewServer(proxyintake.New(relay, refusals))
	defer server.Close()

	response := proxiedResponseTo(t, server, "POST http://site.example/page HTTP/1.1")
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusMethodNotAllowed || len(relay.addresses) != 0 ||
		len(refusals.methods) != 1 || refusals.methods[0] != http.MethodPost {
		t.Fatalf("status %d, addresses %v, refused methods %v",
			response.StatusCode, relay.addresses, refusals.methods)
	}
}

func TestATargetThatIsNotAnAbsoluteWebAddressIsABadRequest(t *testing.T) {
	for _, target := range []string{"/page", "ftp://site.example/page"} {
		relay := &scriptedRelay{}
		refusals := &refusalRecord{}
		server := httptest.NewServer(proxyintake.New(relay, refusals))

		response := proxiedResponseTo(t, server, "GET "+target+" HTTP/1.1")
		_ = response.Body.Close()
		server.Close()

		if response.StatusCode != http.StatusBadRequest || len(relay.addresses) != 0 ||
			len(refusals.targets) != 1 || refusals.targets[0] != target {
			t.Fatalf("%s: status %d, addresses %v, refused targets %v",
				target, response.StatusCode, relay.addresses, refusals.targets)
		}
	}
}

func TestAReplyCutShortEndsTheAnswerEarly(t *testing.T) {
	relay := &scriptedRelay{cutsShort: true}
	server := httptest.NewServer(proxyintake.New(relay, &refusalRecord{}))
	defer server.Close()

	response := proxiedResponseTo(t, server, "GET http://site.example/page HTTP/1.1")
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)

	if string(body) != "page" || err == nil {
		t.Fatalf("body %q, error %v", body, err)
	}
}

func proxiedResponseTo(t *testing.T, server *httptest.Server, requestLine string) *http.Response {
	t.Helper()
	var dialer net.Dialer
	connection, err := dialer.DialContext(
		t.Context(),
		"tcp",
		strings.TrimPrefix(server.URL, "http://"),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.Close() })
	if _, err := io.WriteString(
		connection,
		requestLine+"\r\nHost: site.example\r\n\r\n",
	); err != nil {
		t.Fatal(err)
	}
	response, err := http.ReadResponse(bufio.NewReader(connection), nil)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
