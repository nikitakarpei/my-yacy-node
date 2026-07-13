package cdprender

import (
	"context"
	"fmt"
	"mime"
	"strings"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func extractDocumentBody(
	ctx context.Context,
	requestID network.RequestID,
	contentType string,
) ([]byte, error) {
	if isHypertext(contentType) {
		var html string
		if err := chromedp.Run(
			ctx,
			chromedp.OuterHTML("html", &html, chromedp.ByQuery),
		); err != nil {
			return nil, fmt.Errorf("serialize rendered document: %w", err)
		}
		return []byte(html), nil
	}

	var body []byte
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		raw, err := network.GetResponseBody(requestID).Do(ctx)
		if err != nil {
			return fmt.Errorf("get response body: %w", err)
		}
		body = raw
		return nil
	}))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
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
