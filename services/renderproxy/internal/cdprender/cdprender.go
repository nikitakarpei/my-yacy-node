// Package cdprender renders pages by driving a browser over the Chrome DevTools Protocol.
package cdprender

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/cdpdocumentbinding"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagefreshness"
	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

type Renderer struct {
	allocatorCtx context.Context
	allocatorEnd context.CancelFunc
	maxBytes     int64
}

func New(ctx context.Context, cdpURL string, maxBytes int64) *Renderer {
	allocatorCtx, allocatorEnd := chromedp.NewRemoteAllocator(ctx, cdpURL)
	return &Renderer{allocatorCtx: allocatorCtx, allocatorEnd: allocatorEnd, maxBytes: maxBytes}
}

func (r *Renderer) Close() {
	r.allocatorEnd()
}

func (r *Renderer) Render(
	ctx context.Context,
	target renderedpage.Target,
) (renderedpage.Page, error) {
	tabCtx, tabCancel := chromedp.NewContext(r.allocatorCtx)
	defer tabCancel()

	stopPropagation := context.AfterFunc(ctx, tabCancel)
	defer stopPropagation()

	binding := cdpdocumentbinding.New(ctx)
	chromedp.ListenTarget(tabCtx, binding.Observe)

	if err := chromedp.Run(tabCtx, chromedp.Navigate(target.URL)); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return renderedpage.Page{}, fmt.Errorf("render %s: %w", target.URL, ctxErr)
		}
		return renderedpage.Page{}, fmt.Errorf("render %s: %w", target.URL, err)
	}

	document := binding.BoundDocument()
	if !document.Seen {
		return renderedpage.Page{}, fmt.Errorf(
			"render %s: no document response observed",
			target.URL,
		)
	}

	body, err := serializedDocumentWithinLimit(tabCtx, r.maxBytes)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return renderedpage.Page{}, fmt.Errorf("render %s: %w", target.URL, ctxErr)
		}
		return renderedpage.Page{}, fmt.Errorf("render %s: %w", target.URL, err)
	}

	return renderedpage.Page{
		StatusCode:  document.StatusCode,
		ContentType: document.ContentType,
		ReuseTerms:  pagefreshness.ReuseTermsOf(document.ResponseHeader),
		Body:        body,
	}, nil
}
