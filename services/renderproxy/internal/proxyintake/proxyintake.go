// Package proxyintake accepts absolute-URL forward-proxy requests and refuses tunnels.
package proxyintake

import (
	"context"
	"errors"
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagefreshness"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

const (
	headerContentType = "Content-Type"
	headerLocation    = "Location"
)

type ProxyEndpoint struct {
	renderer renderedpage.Renderer
}

func New(renderer renderedpage.Renderer) *ProxyEndpoint {
	return &ProxyEndpoint{renderer: renderer}
}

func (e *ProxyEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !r.URL.IsAbs() {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	page, err := e.renderer.Render(r.Context(), renderedpage.Target{
		URL:        r.URL.String(),
		Conditions: pagefreshness.ConditionsOf(r.Header),
	})
	if err != nil {
		writeFailure(w, err)
		return
	}

	if page.ContentType != "" {
		w.Header().Set(headerContentType, page.ContentType)
	}
	if page.Location != "" {
		w.Header().Set(headerLocation, page.Location)
	}
	page.ReuseTerms.StateOn(w.Header())
	w.WriteHeader(page.StatusCode)
	if _, err := w.Write(page.Body); err != nil {
		return
	}
}

func writeFailure(w http.ResponseWriter, err error) {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		w.WriteHeader(http.StatusGatewayTimeout)
		return
	}
	w.WriteHeader(http.StatusBadGateway)
}
