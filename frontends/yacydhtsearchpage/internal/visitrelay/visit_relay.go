// Package visitrelay passes a visit that reaches the search page on to visitcrawl, and passes
// the answer of visitcrawl back to the browser. A visit that visitcrawl does not answer gets
// a bad gateway answer.
package visitrelay

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type RelayObserver interface {
	VisitRelayFailed(ctx context.Context, err error)
}

func New(visitcrawlURL *url.URL, observer RelayObserver) http.Handler {
	return &httputil.ReverseProxy{
		Rewrite: func(visit *httputil.ProxyRequest) {
			visit.SetURL(visitcrawlURL)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			observer.VisitRelayFailed(r.Context(), err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		},
	}
}
