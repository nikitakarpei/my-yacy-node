// Package visitrelay passes each visit that a person opens on to visitcrawl and gives the person
// visitcrawl's answer, so the search page and its visit links share one address. A visit that
// cannot reach visitcrawl is reported to the observer and answered as a bad gateway.
package visitrelay

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Observer interface {
	RelayFailed(ctx context.Context, err error)
}

func New(
	visitCrawlURL *url.URL,
	dial func(ctx context.Context, network, address string) (net.Conn, error),
	observer Observer,
) http.Handler {
	return &httputil.ReverseProxy{
		Transport: &http.Transport{DialContext: dial},
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(visitCrawlURL)
		},
		ErrorHandler: func(w http.ResponseWriter, request *http.Request, err error) {
			observer.RelayFailed(request.Context(), err)
			w.WriteHeader(http.StatusBadGateway)
		},
	}
}
