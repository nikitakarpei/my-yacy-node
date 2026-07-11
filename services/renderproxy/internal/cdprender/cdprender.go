// Package cdprender renders pages by driving a browser over the Chrome DevTools Protocol.
package cdprender

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

type Renderer struct {
	allocatorCtx context.Context
	allocatorEnd context.CancelFunc
}

func New(ctx context.Context, cdpURL string) *Renderer {
	allocatorCtx, allocatorEnd := chromedp.NewRemoteAllocator(ctx, cdpURL)
	return &Renderer{allocatorCtx: allocatorCtx, allocatorEnd: allocatorEnd}
}

func (r *Renderer) Close() {
	r.allocatorEnd()
}

func (r *Renderer) Render(ctx context.Context, targetURL string) (renderedpage.Page, error) {
	tabCtx, tabCancel := chromedp.NewContext(r.allocatorCtx)
	defer tabCancel()

	var outcome mainDocumentResponse
	chromedp.ListenTarget(tabCtx, outcome.observe)

	var body string
	if err := chromedp.Run(tabCtx,
		chromedp.Navigate(targetURL),
		chromedp.OuterHTML("html", &body, chromedp.ByQuery),
	); err != nil {
		return renderedpage.Page{}, fmt.Errorf("render %s: %w", targetURL, err)
	}

	statusCode, contentType, ok := outcome.result()
	if !ok {
		return renderedpage.Page{}, fmt.Errorf(
			"render %s: no document response observed",
			targetURL,
		)
	}

	return renderedpage.Page{
		StatusCode:  statusCode,
		ContentType: contentType,
		Body:        []byte(body),
	}, nil
}
