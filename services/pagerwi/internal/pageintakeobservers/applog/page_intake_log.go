// Package applog writes each page intake fact to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageOffered              = "offered page received"
	msgOfferedPageInvalid       = "offered page invalid, no rwi built"
	msgDocumentExtractionFailed = "offered page document extraction failed, no rwi built"
	msgMediaTypeUnsupported     = "offered page has a media type the service does not read, " +
		"no rwi built"
	msgNoIndexDerived      = "offered page derives no index, no rwi built"
	msgPageRWINotPublished = "rwi not published, the offered page waits for a retry"
	msgPageRWIBuilt        = "rwi built and published"
)

type PageIntakeLog struct{}

func (PageIntakeLog) OfferedPageInvalid(ctx context.Context) {
	slog.WarnContext(ctx, msgOfferedPageInvalid)
}

func (PageIntakeLog) PageOffered(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgPageOffered,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (PageIntakeLog) DocumentExtractionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgDocumentExtractionFailed,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageIntakeLog) MediaTypeUnsupported(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	contentType string,
) {
	slog.DebugContext(ctx, msgMediaTypeUnsupported,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.String("contentType", contentType),
	)
}

func (PageIntakeLog) NoIndexDerived(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgNoIndexDerived,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (PageIntakeLog) PageRWINotPublished(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.WarnContext(ctx, msgPageRWINotPublished,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (PageIntakeLog) PageRWIBuilt(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	slog.DebugContext(ctx, msgPageRWIBuilt,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", amountOfPostings),
	)
}
