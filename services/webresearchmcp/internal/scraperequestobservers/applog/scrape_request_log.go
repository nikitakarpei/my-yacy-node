// Package applog writes a log line for each scrape request the service publishes, and for
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
		"the caller reads what the corpus already holds"
	msgScrapeRequestNotPublished = "scrape request not published, " +
		"the caller reads what the corpus already holds"
)

type ScrapeRequestLog struct{}

func (ScrapeRequestLog) ScrapeRequestPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgScrapeRequestPublished, slog.String("pageUrl", pageURL.String()))
}

func (ScrapeRequestLog) ScrapeRequestMarshalingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeRequestUnwritable,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeRequestLog) ScrapeRequestPublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeRequestNotPublished,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}
