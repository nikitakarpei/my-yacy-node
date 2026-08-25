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

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagefreshness"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagereplay"
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

func (r *Renderer) Render(
	ctx context.Context,
	target renderedpage.Target,
) (renderedpage.Page, error) {
	response, err := r.originResponseFor(ctx, target)
	if err != nil {
		return renderedpage.Page{}, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode == http.StatusNotModified {
		return unchangedPageFrom(response), nil
	}
	if isRedirect(response.StatusCode) && response.Header.Get(headerLocation) != "" {
		return redirectPageFrom(response), nil
	}
	if !isHypertext(response.Header.Get(headerContentType)) {
		page, err := r.cappedPageFrom(response)
		if err != nil {
			return renderedpage.Page{}, fmt.Errorf("preflight %s: %w", target.URL, err)
		}
		return page, nil
	}
	return r.inner.Render(ctx, target)
}

func (r *Renderer) originResponseFor(
	ctx context.Context,
	target renderedpage.Target,
) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("preflight %s: %w", target.URL, err)
	}
	target.Conditions.StateOn(request.Header)

	response, err := r.origin.Do(request)
	if err != nil {
		return nil, fmt.Errorf("preflight %s: %w", target.URL, err)
	}
	return response, nil
}

func unchangedPageFrom(response *http.Response) renderedpage.Page {
	return renderedpage.Page{
		StatusCode:   response.StatusCode,
		ReuseTerms:   pagefreshness.ReuseTermsOf(response.Header),
		CaptureTerms: pagereplay.CaptureTermsOf(response.Header),
	}
}

func isRedirect(statusCode int) bool {
	return statusCode >= http.StatusMultipleChoices && statusCode < http.StatusBadRequest
}

func redirectPageFrom(response *http.Response) renderedpage.Page {
	return renderedpage.Page{
		StatusCode:   response.StatusCode,
		ContentType:  response.Header.Get(headerContentType),
		Location:     response.Header.Get(headerLocation),
		ReuseTerms:   pagefreshness.ReuseTermsOf(response.Header),
		CaptureTerms: pagereplay.CaptureTermsOf(response.Header),
	}
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
		StatusCode:   response.StatusCode,
		ContentType:  response.Header.Get(headerContentType),
		ReuseTerms:   pagefreshness.ReuseTermsOf(response.Header),
		CaptureTerms: pagereplay.CaptureTermsOf(response.Header),
		Body:         body,
	}, nil
}
