package verdictadding

import (
	"mime"
	"net/http"
)

type SkipReason string

const (
	HeadRequest SkipReason = "head_request"
	Not2xx      SkipReason = "not_2xx"
	NotHTML     SkipReason = "not_html"
)

var htmlMediaTypes = map[string]bool{"text/html": true, "application/xhtml+xml": true}

func skipReasonOf(method string, upstreamResponse *http.Response) (SkipReason, bool) {
	if method == http.MethodHead {
		return HeadRequest, true
	}
	if upstreamResponse.StatusCode < http.StatusOK ||
		upstreamResponse.StatusCode >= http.StatusMultipleChoices {
		return Not2xx, true
	}
	if mediaType, _, _ := mime.ParseMediaType(
		upstreamResponse.Header.Get("Content-Type"),
	); !htmlMediaTypes[mediaType] {
		return NotHTML, true
	}
	return "", false
}
