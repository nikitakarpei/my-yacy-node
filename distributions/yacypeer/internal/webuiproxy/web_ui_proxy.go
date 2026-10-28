// Package webuiproxy forwards a request that reaches the operator address of the peer to the
// web UI that serves it, with the path and the host that the browser asked for.
package webuiproxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const (
	msgWebUIUnanswered   = "web UI did not answer"
	forwardedProtoHeader = "X-Forwarded-Proto"
)

func To(webUI *url.URL) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(webUI)
			request.Out.Host = request.In.Host
			request.Out.Header[forwardedProtoHeader] = request.In.Header[forwardedProtoHeader]
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			slog.WarnContext(r.Context(), msgWebUIUnanswered,
				slog.String("webUiUrl", webUI.String()),
				slog.String("path", r.URL.Path),
				slog.Any("error", err),
			)
			w.WriteHeader(http.StatusBadGateway)
		},
	}
}
