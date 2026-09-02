package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	msgScrapeRequestInvalid   = "scrape request unreadable, intake halted"
	msgScrapeRequestReceived  = "scrape request received"
	msgOriginReadFailed       = "page read from the origin failed"
	msgPageOffered            = "page offered to the corpora"
	msgPageNotOffered         = "nothing offered to the corpora for this page, the request comes back"
	msgRedirectionNotRecorded = "redirection not recorded, the request comes back"
	msgScrapeDeferred         = "scrape deferred by the origin, scheduled for a later read"
	msgScrapeScheduleFailed   = "scrape not scheduled, the request comes back"
	msgScrapeFailed           = "scrape failed, the page is given up"
	msgScrapeOutcomeAnnounce  = "scrape failure not announced, " +
		"a caller waiting for this page learns nothing until it stops waiting"
)

type ScrapeProgressLog struct{}

func (ScrapeProgressLog) ScrapeRequestInvalid(ctx context.Context, message string, cause error) {
	slog.ErrorContext(ctx, msgScrapeRequestInvalid,
		slog.String("message", message),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) ScrapeRequestReceived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgScrapeRequestReceived, slog.String("pageUrl", pageURL.String()))
}

func (ScrapeProgressLog) OriginReadFailed(
	ctx context.Context,
	fetchURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgOriginReadFailed,
		slog.String("fetchUrl", fetchURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) PageOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	landedURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgPageOffered,
		slog.String("pageUrl", pageURL.String()),
		slog.String("landedUrl", landedURL.String()),
	)
}

func (ScrapeProgressLog) PageNotOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.ErrorContext(ctx, msgPageNotOffered,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) RedirectionNotRecorded(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.ErrorContext(ctx, msgRedirectionNotRecorded,
		slog.String("requestedUrl", requestedURL.String()),
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) ScrapeDeferred(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	deferFor time.Duration,
) {
	slog.DebugContext(ctx, msgScrapeDeferred,
		slog.String("pageUrl", pageURL.String()),
		slog.Duration("deferFor", deferFor),
	)
}

func (ScrapeProgressLog) ScrapeScheduleFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.ErrorContext(ctx, msgScrapeScheduleFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) ScrapeFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	reason pagescrapecontract.ScrapeFailureReason,
) {
	slog.WarnContext(ctx, msgScrapeFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.String("reason", string(reason)),
	)
}

func (ScrapeProgressLog) ScrapeOutcomeAnnouncementFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeOutcomeAnnounce,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}
