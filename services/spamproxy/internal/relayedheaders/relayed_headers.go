// Package relayedheaders chooses the headers that the proxy passes on in each
// direction.
package relayedheaders

import (
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/contentencoding"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
)

const (
	acceptEncodingName = "Accept-Encoding"
	userAgentName      = "User-Agent"
)

var forwardedRequestHeaderNames = []string{userAgentName, "If-None-Match", "If-Modified-Since"}

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

func ForwardedHeadersFrom(requestHeaders http.Header) http.Header {
	forwardedHeaders := http.Header{userAgentName: nil}
	for _, name := range forwardedRequestHeaderNames {
		for _, value := range requestHeaders.Values(name) {
			forwardedHeaders.Add(name, value)
		}
	}
	for _, value := range requestHeaders.Values(acceptEncodingName) {
		forwardedHeaders.Add(acceptEncodingName, contentencoding.DecodableAcceptEncodingFrom(value))
	}
	return forwardedHeaders
}

func EndToEndHeadersOf(responseHeaders http.Header) http.Header {
	endToEndHeaders := responseHeaders.Clone()
	for _, value := range responseHeaders.Values("Connection") {
		for name := range strings.SplitSeq(value, ",") {
			endToEndHeaders.Del(strings.TrimSpace(name))
		}
	}
	for _, name := range hopByHopHeaderNames {
		endToEndHeaders.Del(name)
	}
	endToEndHeaders.Del(spamassessment.HeaderName)
	return endToEndHeaders
}
