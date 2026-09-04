// Package applog writes a log line for each crawled page the crawler reports, and for each
// report that never leaves. It is the only place that decides how a fact reads and at which
// level it is written.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
)

const (
	msgCrawledPageReported   = "crawled page reported"
	msgCrawledPageUnwritable = "crawled page cannot be written, " +
		"no consumer ever hears of it"
	msgCrawledPageNotReported = "crawled page not reported, " +
		"no consumer ever hears of it"
)

type CrawledPageLog struct{}

func (CrawledPageLog) CrawledPageReported(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
) {
	slog.DebugContext(ctx, msgCrawledPageReported,
		slog.String("url", pageURL.String()),
		slog.String("indexing", string(indexing)),
	)
}

func (CrawledPageLog) CrawledPageEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
	cause error,
) {
	slog.WarnContext(ctx, msgCrawledPageUnwritable,
		slog.String("url", pageURL.String()),
		slog.String("indexing", string(indexing)),
		slog.Any("error", cause),
	)
}

func (CrawledPageLog) CrawledPageReportingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
	cause error,
) {
	slog.WarnContext(ctx, msgCrawledPageNotReported,
		slog.String("url", pageURL.String()),
		slog.String("indexing", string(indexing)),
		slog.Any("error", cause),
	)
}
