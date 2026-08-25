package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

const (
	headerMementoDatetime  = "Memento-Datetime"
	headerLink             = "Link"
	headerCapturedETag     = "X-Archive-Orig-ETag"
	headerCapturedModified = "X-Archive-Orig-Last-Modified"

	relationParameter = "rel"
	originalRelation  = "original"

	msgOriginalURLMissing  = "replayed page names no original url, page read under its replay url"
	msgOriginalURLRejected = "replayed page names a rejected original url, page read under its replay url"
)

// ReplayedCaptureTransport reads a replayed page under the url it was captured from and
// under the validators the origin gave at capture time, so that a page an archive replays
// is the page the origin served.
type ReplayedCaptureTransport struct {
	inner http.RoundTripper
}

func NewReplayedCaptureTransport(inner http.RoundTripper) *ReplayedCaptureTransport {
	return &ReplayedCaptureTransport{inner: inner}
}

func (t *ReplayedCaptureTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := t.inner.RoundTrip(request)
	if err != nil {
		return nil, fmt.Errorf("round trip %s: %w", request.URL, err)
	}
	if response.Header.Get(headerMementoDatetime) == "" {
		return response, nil
	}
	originalURL, named := originalURLFrom(request, response.Header.Get(headerLink))
	if !named {
		return response, nil
	}
	capturedRequest := *response.Request
	capturedRequest.URL = originalURL
	response.Request = &capturedRequest
	stateCapturedValidatorsOn(response.Header)
	return response, nil
}

func originalURLFrom(request *http.Request, linkHeader string) (*url.URL, bool) {
	target, named := originalTargetFrom(linkHeader)
	if !named {
		slog.WarnContext(request.Context(), msgOriginalURLMissing,
			slog.String("url", request.URL.String()),
		)
		return nil, false
	}
	originalURL, err := url.Parse(target)
	if err != nil {
		slog.WarnContext(request.Context(), msgOriginalURLRejected,
			slog.String("url", request.URL.String()),
			slog.String("originalUrl", target),
			slog.Any("error", err),
		)
		return nil, false
	}
	return originalURL, true
}

func originalTargetFrom(linkHeader string) (string, bool) {
	for remainder := linkHeader; ; {
		opened := strings.Index(remainder, "<")
		if opened < 0 {
			return "", false
		}
		closed := strings.Index(remainder[opened:], ">")
		if closed < 0 {
			return "", false
		}
		closed += opened
		if namesOriginalRelation(parametersAfter(remainder[closed+1:])) {
			return remainder[opened+1 : closed], true
		}
		remainder = remainder[closed+1:]
	}
}

func parametersAfter(remainder string) string {
	if nextTarget := strings.Index(remainder, "<"); nextTarget >= 0 {
		return remainder[:nextTarget]
	}
	return remainder
}

func namesOriginalRelation(parameters string) bool {
	for _, parameter := range strings.Split(parameters, ";") {
		name, value, assigned := strings.Cut(parameter, "=")
		if !assigned || !strings.EqualFold(strings.Trim(name, ` ,`), relationParameter) {
			continue
		}
		if strings.EqualFold(strings.Trim(value, ` ",`), originalRelation) {
			return true
		}
	}
	return false
}

func stateCapturedValidatorsOn(responseHeader http.Header) {
	responseHeader.Del(headerETag)
	if capturedETag := responseHeader.Get(headerCapturedETag); capturedETag != "" {
		responseHeader.Set(headerETag, capturedETag)
	}
	responseHeader.Del(headerLastModified)
	if capturedModified := responseHeader.Get(headerCapturedModified); capturedModified != "" {
		responseHeader.Set(headerLastModified, capturedModified)
	}
}
