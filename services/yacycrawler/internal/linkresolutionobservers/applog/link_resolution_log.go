package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgBaseHrefUnresolved  = "base href unresolved, using page url"
	msgLinkHrefsUnresolved = "link hrefs unresolved, left off the frontier"
)

type LinkResolutionLog struct{}

func (LinkResolutionLog) BaseHrefUnresolved(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	baseHref string,
	cause error,
) {
	slog.WarnContext(ctx, msgBaseHrefUnresolved,
		slog.String("url", pageURL.String()),
		slog.String("baseHref", baseHref),
		slog.Any("error", cause),
	)
}

func (LinkResolutionLog) LinkHrefsUnresolved(
	ctx context.Context,
	baseURL canonicalurl.CanonicalURL,
	hrefs int,
) {
	slog.WarnContext(ctx, msgLinkHrefsUnresolved,
		slog.String("url", baseURL.String()),
		slog.Int("hrefs", hrefs),
	)
}
