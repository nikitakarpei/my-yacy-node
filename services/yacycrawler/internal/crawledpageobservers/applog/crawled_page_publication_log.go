// Package applog writes a log line for each crawled page the crawler publishes, and for
// each page that never leaves. It is the only place that decides how a fact reads and at
// which level it is written.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	crawledpagesjetstream "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawledpages/jetstream"
)

const (
	msgCrawledPagePublished  = "crawled page published"
	msgCrawledPageUnwritable = "crawled page cannot be written, " +
		"no consumer ever hears of it"
	msgCrawledPageNotPublished = "crawled page not published, " +
		"no consumer ever hears of it"
)

type CrawledPagePublicationLog struct{}

func (CrawledPagePublicationLog) CrawledPagePublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
) {
	slog.DebugContext(ctx, msgCrawledPagePublished,
		slog.String("url", pageURL.String()),
		slog.String("indexing", string(indexing)),
	)
}

func (CrawledPagePublicationLog) CrawledPageEncodingFailed(
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

func (CrawledPagePublicationLog) CrawledPagePublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing crawledpagesjetstream.PageIndexing,
	cause error,
) {
	slog.WarnContext(ctx, msgCrawledPageNotPublished,
		slog.String("url", pageURL.String()),
		slog.String("indexing", string(indexing)),
		slog.Any("error", cause),
	)
}
