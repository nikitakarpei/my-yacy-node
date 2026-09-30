package relayedheaders_test

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/relayedheaders"
)

func TestOnlyTheNamedRequestHeadersAreRelayed(t *testing.T) {
	requestHeaders := relayedheaders.RequestHeadersFrom(http.Header{
		"User-Agent":        {"crawler"},
		"Cookie":            {"session=1"},
		"If-None-Match":     {`"v1"`},
		"If-Modified-Since": {"Tue, 29 Sep 2026 10:00:00 GMT"},
		"Accept-Encoding":   {"gzip, br"},
	})

	want := http.Header{
		"User-Agent":        {"crawler"},
		"If-None-Match":     {`"v1"`},
		"If-Modified-Since": {"Tue, 29 Sep 2026 10:00:00 GMT"},
		"Accept-Encoding":   {"gzip"},
	}
	if !reflect.DeepEqual(requestHeaders, want) {
		t.Fatalf("request headers = %v, want %v", requestHeaders, want)
	}
}

func TestHopByHopHeadersAreNotRelayed(t *testing.T) {
	responseHeaders := relayedheaders.ResponseHeadersFrom(http.Header{
		"Content-Type":      {"text/html"},
		"Connection":        {"close, X-Private"},
		"X-Private":         {"1"},
		"Keep-Alive":        {"timeout=5"},
		"Transfer-Encoding": {"chunked"},
		"Etag":              {`"v1"`},
	})

	want := http.Header{"Content-Type": {"text/html"}, "Etag": {`"v1"`}}
	if !reflect.DeepEqual(responseHeaders, want) {
		t.Fatalf("response headers = %v, want %v", responseHeaders, want)
	}
}

func TestARequestWithoutAUserAgentIsSentWithoutOne(t *testing.T) {
	request := &http.Request{
		Method: http.MethodGet,
		URL:    &url.URL{Scheme: "http", Host: "site.example", Path: "/"},
		Header: relayedheaders.RequestHeadersFrom(http.Header{}),
	}
	var sentRequest strings.Builder

	if err := request.Write(&sentRequest); err != nil {
		t.Fatal(err)
	}

	if strings.Contains(sentRequest.String(), "User-Agent") {
		t.Fatalf("sent request:\n%s", sentRequest.String())
	}
}
