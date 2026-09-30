// Package relayedheaders chooses the headers that the proxy passes on in each
// direction.
package relayedheaders

import (
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
)

const (
	acceptEncodingName = "Accept-Encoding"
	userAgentName      = "User-Agent"
)

var relayedRequestHeaderNames = []string{userAgentName, "If-None-Match", "If-Modified-Since"}

var hopByHopHeaderNames = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func RequestHeadersFrom(clientRequestHeaders http.Header) http.Header {
	requestHeaders := http.Header{userAgentName: nil}
	for _, name := range relayedRequestHeaderNames {
		for _, value := range clientRequestHeaders.Values(name) {
			requestHeaders.Add(name, value)
		}
	}
	for _, value := range clientRequestHeaders.Values(acceptEncodingName) {
		requestHeaders.Add(acceptEncodingName, contentencoding.DecodableAcceptEncodingFrom(value))
	}
	return requestHeaders
}

func ResponseHeadersFrom(upstreamResponseHeaders http.Header) http.Header {
	responseHeaders := upstreamResponseHeaders.Clone()
	for _, value := range upstreamResponseHeaders.Values("Connection") {
		for name := range strings.SplitSeq(value, ",") {
			responseHeaders.Del(strings.TrimSpace(name))
		}
	}
	for _, name := range hopByHopHeaderNames {
		responseHeaders.Del(name)
	}
	return responseHeaders
}
