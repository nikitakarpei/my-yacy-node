package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgIndexingRefusalEnforced      = "page indexing refusal enforced"
	msgLinkDiscoveryRefusalEnforced = "page link discovery refusal enforced"
	msgPageHTMLUnreadable           = "page html unreadable"
)

type PageReadingLog struct{}

func (PageReadingLog) IndexingRefusalEnforced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgIndexingRefusalEnforced, slog.String("url", pageURL.String()))
}

func (PageReadingLog) LinkDiscoveryRefusalEnforced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgLinkDiscoveryRefusalEnforced, slog.String("url", pageURL.String()))
}

func (PageReadingLog) PageHTMLUnreadable(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgPageHTMLUnreadable,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}
