package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgMediaTypeUnparsed            = "content type unparsed, falling back to its leading segment"
	msgLinkDiscoveryRefusalEnforced = "page link discovery refusal enforced"
)

type PageReadingLog struct{}

func (PageReadingLog) MediaTypeUnparsed(
	ctx context.Context,
	contentType string,
	cause error,
) {
	slog.WarnContext(ctx, msgMediaTypeUnparsed,
		slog.String("contentType", contentType),
		slog.Any("error", cause),
	)
}

func (PageReadingLog) LinkDiscoveryRefusalEnforced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgLinkDiscoveryRefusalEnforced, slog.String("url", pageURL.String()))
}
