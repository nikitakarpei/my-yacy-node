package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	msgScrapeRequestInvalid  = "scrape request unreadable, intake halted"
	msgScrapeRequestReceived = "scrape request received"
	msgOriginFetchFailed     = "page fetch from the origin failed"
	msgPageOffered           = "page offered to the corpora"
	msgPageNotOffered        = "nothing offered to the corpora for this page, the request comes back"
	msgScrapeDeferred        = "scrape deferred by the origin, scheduled for a later fetch"
	msgScrapeScheduleFailed  = "scrape not scheduled, the request comes back"
	msgScrapeFailed          = "scrape failed, the page is given up"
)

type ScrapeIntakeLog struct{}

func (ScrapeIntakeLog) ScrapeRequestInvalid(ctx context.Context, message string, cause error) {
	slog.ErrorContext(ctx, msgScrapeRequestInvalid,
		slog.String("message", message),
		slog.Any("error", cause),
	)
}

func (ScrapeIntakeLog) ScrapeRequestReceived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgScrapeRequestReceived, slog.String("pageUrl", pageURL.String()))
}

func (ScrapeIntakeLog) OriginFetchFailed(
	ctx context.Context,
	fetchURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgOriginFetchFailed,
		slog.String("fetchUrl", fetchURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeIntakeLog) PageOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	landedURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgPageOffered,
		slog.String("pageUrl", pageURL.String()),
		slog.String("landedUrl", landedURL.String()),
	)
}

func (ScrapeIntakeLog) PageNotOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.ErrorContext(ctx, msgPageNotOffered,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeIntakeLog) ScrapeDeferred(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	deferFor time.Duration,
) {
	slog.DebugContext(ctx, msgScrapeDeferred,
		slog.String("pageUrl", pageURL.String()),
		slog.Duration("deferFor", deferFor),
	)
}

func (ScrapeIntakeLog) ScrapeScheduleFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.ErrorContext(ctx, msgScrapeScheduleFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeIntakeLog) ScrapeFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	reason pagescrapecontract.ScrapeFailureReason,
) {
	slog.WarnContext(ctx, msgScrapeFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.String("reason", string(reason)),
	)
}
