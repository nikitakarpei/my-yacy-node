// Package originpreflight answers from the origin's own response: it relays a redirect,
// serves a body that needs no rendering, and hands a hypertext page to the browser.
package originpreflight

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

const (
	headerLocation    = "Location"
	headerContentType = "Content-Type"
)

type Renderer struct {
	origin   *http.Client
	inner    renderedpage.Renderer
	maxBytes int64
}

func New(inner renderedpage.Renderer, egressProxy *url.URL, maxBytes int64) *Renderer {
	return &Renderer{
		origin: &http.Client{
			Transport: &http.Transport{Proxy: http.ProxyURL(egressProxy)},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		inner:    inner,
		maxBytes: maxBytes,
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
			ContentType: response.Header.Get(headerContentType),
			Location:    location,
		}, nil
	}
	if !isHypertext(response.Header.Get(headerContentType)) {
		page, err := r.cappedPageFrom(response)
		if err != nil {
			return renderedpage.Page{}, fmt.Errorf("preflight %s: %w", targetURL, err)
		}
		return page, nil
	}
	return r.inner.Render(ctx, targetURL)
}

func isRedirect(statusCode int) bool {
	return statusCode >= http.StatusMultipleChoices && statusCode < http.StatusBadRequest
}

func isHypertext(contentType string) bool {
	essence, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		essence = strings.TrimSpace(contentType)
	}
	switch strings.ToLower(essence) {
	case "text/html", "application/xhtml+xml":
		return true
	default:
		return false
	}
}

func (r *Renderer) cappedPageFrom(response *http.Response) (renderedpage.Page, error) {
	body, err := io.ReadAll(io.LimitReader(response.Body, r.maxBytes+1))
	if err != nil {
		return renderedpage.Page{}, fmt.Errorf("read origin body: %w", err)
	}
	if int64(len(body)) > r.maxBytes {
		return renderedpage.Page{}, fmt.Errorf(
			"%w: origin body exceeds limit %d", renderedpage.ErrTooLarge, r.maxBytes,
		)
	}
	return renderedpage.Page{
		StatusCode:  response.StatusCode,
		ContentType: response.Header.Get(headerContentType),
		Body:        body,
	}, nil
}
