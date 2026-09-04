// Package applog writes a log line for each fact the text corpus learns about a page the
// scrape service offered it, so an operator can read the whole life of one page in the
// service log. It is the only place that decides how a fact reads and at which level it is
// written.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageOffered      = "offered page received"
	msgExtractionFailed = "offered page document extraction failed, nothing indexed"
	msgNoTextDerived    = "offered page derives no text, nothing indexed"
	msgIndexFailed      = "offered page index failed"
	msgPageIndexed      = "offered page indexed"
)

type IntakeProgressLog struct{}

func (IntakeProgressLog) PageOffered(ctx context.Context, pageURL canonicalurl.CanonicalURL) {
	slog.DebugContext(ctx, msgPageOffered, slog.String("pageUrl", pageURL.String()))
}

func (IntakeProgressLog) NoDocumentExtracted(
	ctx context.Context,
	landedURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgExtractionFailed,
		slog.String("landedUrl", landedURL.String()),
		slog.Any("error", cause),
	)
}

func (IntakeProgressLog) NoReadableTextDerived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgNoTextDerived, slog.String("pageUrl", pageURL.String()))
}

func (IntakeProgressLog) IndexObserved(context.Context, time.Duration) {}

func (IntakeProgressLog) IndexFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgIndexFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (IntakeProgressLog) PageIndexed(ctx context.Context, pageURL canonicalurl.CanonicalURL) {
	slog.DebugContext(ctx, msgPageIndexed, slog.String("pageUrl", pageURL.String()))
}
