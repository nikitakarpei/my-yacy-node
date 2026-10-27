// Package visitrelay passes each visit that a person opens on to visitcrawl and gives the person
// visitcrawl's answer, so the search page and its visit links share one address. A visit that
// cannot reach visitcrawl is reported to the observer and answered as a bad gateway.
package visitrelay

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Observer interface {
	RelayFailed(ctx context.Context, err error)
}

func New(visitCrawlURL *url.URL, observer Observer) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			request.SetURL(visitCrawlURL)
		},
		ErrorHandler: func(w http.ResponseWriter, request *http.Request, err error) {
			observer.RelayFailed(request.Context(), err)
			w.WriteHeader(http.StatusBadGateway)
		},
	}
}
