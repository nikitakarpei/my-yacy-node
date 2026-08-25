// Package applog writes a log line for each fact the markdown corpus learns about a scrape
// request it took on, so an operator can read the whole life of one request in the service
// log. It is the only place that decides how a fact reads and at which level it is written.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgScrapeRequestReceived        = "scrape request received"
	msgOriginFetchFailed            = "scrape request fetch failed"
	msgOriginFetchDeferred          = "scrape request fetch deferred by the origin"
	msgNothingToScrape              = "scrape request fetch holds no content to scrape"
	msgDocumentExtractionFailed     = "scrape request document extraction failed, nothing stored"
	msgNoMarkdownDerived            = "scrape request derives no markdown, nothing stored"
	msgMarkdownCorpusWriteFailed    = "page markdown store failed"
	msgRedirectionRecordWriteFailed = "page redirection not recorded, " +
		"recall by the requested url would miss"
	msgMarkdownStored = "page markdown stored"
)

type ScrapeProgressLog struct{}

func (ScrapeProgressLog) ScrapeRequestReceived(ctx context.Context) {
	slog.DebugContext(ctx, msgScrapeRequestReceived)
}

func (ScrapeProgressLog) OriginFetchFailed(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgOriginFetchFailed,
		slog.String("requestedUrl", requestedURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) OriginFetchDeferred(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	deferFor time.Duration,
) {
	slog.DebugContext(ctx, msgOriginFetchDeferred,
		slog.String("requestedUrl", requestedURL.String()),
		slog.Duration("deferFor", deferFor),
	)
}

func (ScrapeProgressLog) NothingToScrape(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgNothingToScrape,
		slog.String("requestedUrl", requestedURL.String()),
	)
}

func (ScrapeProgressLog) DocumentExtractionFailed(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgDocumentExtractionFailed,
		slog.String("requestedUrl", requestedURL.String()),
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) NoMarkdownDerived(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgNoMarkdownDerived,
		slog.String("requestedUrl", requestedURL.String()),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (ScrapeProgressLog) MarkdownCorpusWriteFailed(
	ctx context.Context,
	markdownURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgMarkdownCorpusWriteFailed,
		slog.String("markdownUrl", markdownURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) RedirectionRecordWriteFailed(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	markdownURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgRedirectionRecordWriteFailed,
		slog.String("requestedUrl", requestedURL.String()),
		slog.String("markdownUrl", markdownURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) MarkdownStored(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	markdownURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgMarkdownStored,
		slog.String("requestedUrl", requestedURL.String()),
		slog.String("markdownUrl", markdownURL.String()),
	)
}
