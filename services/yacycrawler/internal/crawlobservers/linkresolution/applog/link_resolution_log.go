package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgBaseURLUnresolved = "base url unresolved, using the enclosing base url"
	msgLinksUnresolved   = "links unresolved, left off the frontier"
)

type LinkResolutionLog struct{}

func (LinkResolutionLog) BaseURLUnresolved(
	ctx context.Context,
	enclosingBaseURL canonicalurl.CanonicalURL,
	statedBaseURL string,
	cause error,
) {
	slog.WarnContext(ctx, msgBaseURLUnresolved,
		slog.String("url", enclosingBaseURL.String()),
		slog.String("statedBaseUrl", statedBaseURL),
		slog.Any("error", cause),
	)
}

func (LinkResolutionLog) LinksUnresolved(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	links int,
) {
	slog.WarnContext(ctx, msgLinksUnresolved,
		slog.String("url", pageURL.String()),
		slog.Int("links", links),
	)
}
