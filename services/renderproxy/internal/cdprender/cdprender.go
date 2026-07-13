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

	stopPropagation := context.AfterFunc(ctx, tabCancel)
	defer stopPropagation()

	var outcome mainDocumentResponse
	chromedp.ListenTarget(tabCtx, outcome.observe)

	if err := chromedp.Run(tabCtx, chromedp.Navigate(targetURL)); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return renderedpage.Page{}, fmt.Errorf("render %s: %w", targetURL, ctxErr)
		}
		return renderedpage.Page{}, fmt.Errorf("render %s: %w", targetURL, err)
	}

	document := outcome.result()
	if !document.seen {
		return renderedpage.Page{}, fmt.Errorf(
			"render %s: no document response observed",
			targetURL,
		)
	}

	body, err := extractDocumentBody(tabCtx, document.requestID, document.contentType)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return renderedpage.Page{}, fmt.Errorf("render %s: %w", targetURL, ctxErr)
		}
		return renderedpage.Page{}, fmt.Errorf("render %s: %w", targetURL, err)
	}

	return renderedpage.Page{
		StatusCode:  document.statusCode,
		ContentType: document.contentType,
		Body:        body,
	}, nil
}
