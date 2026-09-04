// Package applog writes a log line for each fact the markdown corpus learns about a page the
// scrape service offered it, so an operator can read the whole life of one page in the
// service log. It is the only place that decides how a fact reads and at which level it is
// written.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageOffered       = "offered page received"
	msgExtractionFailed  = "offered page document extraction failed, nothing stored"
	msgNoMarkdownDerived = "offered page derives no markdown, nothing stored"
	msgMarkdownNotStored = "offered page markdown write failed"
	msgMarkdownStored    = "offered page markdown stored"
)

type PageIntakeLog struct{}

func (PageIntakeLog) PageOffered(ctx context.Context, pageURL canonicalurl.CanonicalURL) {
	slog.DebugContext(ctx, msgPageOffered, slog.String("pageUrl", pageURL.String()))
}

func (PageIntakeLog) NoDocumentExtracted(
	ctx context.Context,
	landedURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgExtractionFailed,
		slog.String("landedUrl", landedURL.String()),
		slog.Any("error", cause),
	)
}

func (PageIntakeLog) NoMarkdownDerived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgNoMarkdownDerived, slog.String("pageUrl", pageURL.String()))
}

func (PageIntakeLog) MarkdownNotStored(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgMarkdownNotStored,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageIntakeLog) MarkdownStored(ctx context.Context, pageURL canonicalurl.CanonicalURL) {
	slog.DebugContext(ctx, msgMarkdownStored, slog.String("pageUrl", pageURL.String()))
}
