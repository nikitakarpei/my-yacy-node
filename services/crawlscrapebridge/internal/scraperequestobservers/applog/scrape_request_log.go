// Package applog writes a log line for each scrape request the bridge publishes, and for each
// one that does not leave, so an operator can see why a crawled page was not scraped yet.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgScrapeRequestPublished    = "scrape request published"
	msgScrapeRequestNotPublished = "scrape request not published, the crawled page is relayed again later"
)

type ScrapeRequestLog struct{}

func (ScrapeRequestLog) ScrapeRequestPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgScrapeRequestPublished, slog.String("pageUrl", pageURL.String()))
}

func (ScrapeRequestLog) ScrapeRequestNotPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeRequestNotPublished,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}
