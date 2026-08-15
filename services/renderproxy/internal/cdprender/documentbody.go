package cdprender

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/renderedpage"
)

func serializedDocumentWithinLimit(ctx context.Context, maxBytes int64) ([]byte, error) {
	sizeInBrowser, err := documentSizeInBrowser(ctx)
	if err != nil {
		return nil, err
	}
	if sizeInBrowser > maxBytes {
		return nil, fmt.Errorf(
			"%w: rendered document is %d bytes, limit %d",
			renderedpage.ErrTooLarge, sizeInBrowser, maxBytes,
		)
	}

	var html string
	if err := chromedp.Run(
		ctx,
		chromedp.OuterHTML("html", &html, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("serialize rendered document: %w", err)
	}
	return []byte(html), nil
}

func documentSizeInBrowser(ctx context.Context) (int64, error) {
	var byteCount int64
	if err := chromedp.Run(ctx, chromedp.Evaluate(
		`unescape(encodeURIComponent(document.documentElement.outerHTML)).length`,
		&byteCount,
	)); err != nil {
		return 0, fmt.Errorf("measure rendered document: %w", err)
	}
	return byteCount, nil
}
