// Package redirectpreflight relays an origin redirect verbatim before falling back to a browser render.
package redirectpreflight

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

const headerLocation = "Location"

type Renderer struct {
	origin *http.Client
	inner  renderedpage.Renderer
}

func New(inner renderedpage.Renderer, egressProxy *url.URL) *Renderer {
	return &Renderer{
		origin: &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(egressProxy)},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		inner: inner,
	}
}

func (r *Renderer) Render(ctx context.Context, targetURL string) (renderedpage.Page, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return renderedpage.Page{}, fmt.Errorf("preflight %s: %w", targetURL, err)
	}
	response, err := r.origin.Do(request)
	if err != nil {
		return renderedpage.Page{}, fmt.Errorf("preflight %s: %w", targetURL, err)
	}
	defer func() { _ = response.Body.Close() }()

	location := response.Header.Get(headerLocation)
	if isRedirect(response.StatusCode) && location != "" {
		return renderedpage.Page{
			StatusCode:  response.StatusCode,
			ContentType: response.Header.Get("Content-Type"),
			Location:    location,
		}, nil
	}
	return r.inner.Render(ctx, targetURL)
}

func isRedirect(statusCode int) bool {
	return statusCode >= http.StatusMultipleChoices && statusCode < http.StatusBadRequest
}
