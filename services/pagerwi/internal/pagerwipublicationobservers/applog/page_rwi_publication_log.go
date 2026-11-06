// Package applog writes a log line for each RWI the service publishes for the nodes, and for
// each RWI the stream does not take.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageRWIPublished        = "rwi published"
	msgPageRWIEncodingFailed   = "rwi cannot be written"
	msgPageRWIPublishingFailed = "rwi not taken by the stream"
)

type PageRWIPublicationLog struct{}

func (PageRWIPublicationLog) PageRWIPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgPageRWIPublished, slog.String("pageUrl", pageURL.String()))
}

func (PageRWIPublicationLog) PageRWIEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.ErrorContext(ctx, msgPageRWIEncodingFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageRWIPublicationLog) PageRWIPublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgPageRWIPublishingFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}
