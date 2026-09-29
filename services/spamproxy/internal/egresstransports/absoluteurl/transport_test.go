package absoluteurl_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/egresstransports/absoluteurl"
)

type requestRecord struct {
	requestURIs chan string
	hosts       chan string
}

func newRequestRecord() requestRecord {
	return requestRecord{requestURIs: make(chan string, 1), hosts: make(chan string, 1)}
}

func (r requestRecord) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	r.requestURIs <- request.RequestURI
	r.hosts <- request.Host
	if request.URL.Path == "/stalls" {
		w.WriteHeader(http.StatusOK)
		_ = http.NewResponseController(w).Flush()
		<-request.Context().Done()
		return
	}
	_, _ = io.WriteString(w, "page")
}

func TestAnHTTPSPageIsAskedOfTheProxyByItsAbsoluteURL(t *testing.T) {
	proxy := newRequestRecord()
	server := httptest.NewServer(proxy)
	defer server.Close()

	body := bodyOf(t, server, "https://site.example/page?q=1", t.Context())

	requestURI, host := <-proxy.requestURIs, <-proxy.hosts
	if requestURI != "https://site.example/page?q=1" || host != "site.example" || body != "page" {
		t.Fatalf("request URI %s, host %s, body %q", requestURI, host, body)
	}
}

func TestACancelledRequestStopsReadingTheResponse(t *testing.T) {
	proxy := newRequestRecord()
	server := httptest.NewServer(proxy)
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	response := responseOf(t, server, "http://site.example/stalls", ctx)
	defer func() { _ = response.Body.Close() }()

	cancel()
	_, err := io.ReadAll(response.Body)

	if err == nil {
		t.Fatal("the response read to its end")
	}
}

func TestTheConnectionAndTheWrittenRequestAreTraced(t *testing.T) {
	proxy := newRequestRecord()
	server := httptest.NewServer(proxy)
	defer server.Close()
	var wasConnected, wasRequestWritten bool
	ctx := httptrace.WithClientTrace(t.Context(), &httptrace.ClientTrace{
		GotConn: func(httptrace.GotConnInfo) { wasConnected = true },
		WroteRequest: func(written httptrace.WroteRequestInfo) {
			wasRequestWritten = written.Err == nil
		},
	})

	bodyOf(t, server, "http://site.example/page", ctx)

	if !wasConnected || !wasRequestWritten {
		t.Fatalf("connected %v, request written %v", wasConnected, wasRequestWritten)
	}
}

func TestAnUnreachableProxyFailsTheRequest(t *testing.T) {
	transport := absoluteurl.New(&url.URL{Scheme: "http", Host: "127.0.0.1:1"})
	request, _ := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"https://site.example/",
		nil,
	)

	response, err := transport.RoundTrip(request)
	if err == nil {
		_ = response.Body.Close()
		t.Fatal("the request reached a proxy")
	}
}

func bodyOf(t *testing.T, server *httptest.Server, address string, ctx context.Context) string {
	t.Helper()
	response := responseOf(t, server, address, ctx)
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func responseOf(
	t *testing.T,
	server *httptest.Server,
	address string,
	ctx context.Context,
) *http.Response {
	t.Helper()
	proxyURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := absoluteurl.New(proxyURL).RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
