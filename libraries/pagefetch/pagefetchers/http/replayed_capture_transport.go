package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

const (
	headerMementoDatetime  = "Memento-Datetime"
	headerLink             = "Link"
	headerCapturedETag     = "X-Archive-Orig-ETag"
	headerCapturedModified = "X-Archive-Orig-Last-Modified"

	relationParameter = "rel"
	originalRelation  = "original"

	msgOriginalURLMissing  = "replayed page names no original url, page treated as no page"
	msgOriginalURLRejected = "replayed page names a rejected original url, page treated as no page"
)

type replayedCapture struct {
	OriginalURL   canonicalurl.CanonicalURL
	OriginVersion pagefetch.PageVersion
}

func replayedCaptureOf(
	ctx context.Context,
	response *http.Response,
) (replayedCapture, bool) {
	if response.Header.Get(headerMementoDatetime) == "" {
		return replayedCapture{}, false
	}
	target, named := originalTargetFrom(response.Header.Get(headerLink))
	if !named {
		slog.WarnContext(ctx, msgOriginalURLMissing,
			slog.String("url", response.Request.URL.String()),
		)
		return replayedCapture{}, false
	}
	originalURL, err := canonicalurl.CanonicalURLOf(target)
	if err != nil {
		slog.WarnContext(ctx, msgOriginalURLRejected,
			slog.String("url", response.Request.URL.String()),
			slog.String("originalUrl", target),
			slog.Any("error", err),
		)
		return replayedCapture{}, false
	}
	return replayedCapture{
		OriginalURL: originalURL,
		OriginVersion: pageVersionFrom(
			response.Header.Get(headerCapturedETag),
			response.Header.Get(headerCapturedModified),
		),
	}, true
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
