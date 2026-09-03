package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgScrapeRequestPublished         = "scrape request published"
	msgScrapeRequestPublicationFailed = "scrape request publication failed"
)

type ScrapeRequestLog struct{}

func (ScrapeRequestLog) ScrapeRequestPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgScrapeRequestPublished, slog.String("url", pageURL.String()))
}

func (ScrapeRequestLog) ScrapeRequestPublicationFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeRequestPublicationFailed,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}
