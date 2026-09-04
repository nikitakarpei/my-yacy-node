// Package applog writes a log line for each scrape request the crawler publishes, and for
// each one that never leaves. It is the only place that decides how a fact reads and at
// which level it is written.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgScrapeRequestPublished  = "scrape request published"
	msgScrapeRequestUnwritable = "scrape request cannot be written, " +
		"this page is never scraped"
	msgScrapeRequestNotPublished = "scrape request not published, " +
		"this page is never scraped"
)

type ScrapeRequestLog struct{}

func (ScrapeRequestLog) ScrapeRequestPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgScrapeRequestPublished, slog.String("url", pageURL.String()))
}

func (ScrapeRequestLog) ScrapeRequestEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeRequestUnwritable,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeRequestLog) ScrapeRequestPublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeRequestNotPublished,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}
