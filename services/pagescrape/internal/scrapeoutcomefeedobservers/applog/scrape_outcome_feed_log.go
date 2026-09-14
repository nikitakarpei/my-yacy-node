// Package applog writes a log line for each message the page feed carries to a caller
// waiting on one page, and for each message that never gets there. It is the only place
// that decides how a fact reads and at which level it is written.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgFailureAnnounced  = "scrape failure announced"
	msgFailureUnreadable = "scrape failure cannot be written, " +
		"a caller waiting for this page learns nothing until it stops waiting"
	msgFailureNotAnnounced = "scrape failure not announced, " +
		"a caller waiting for this page learns nothing until it stops waiting"
	msgFailureUnconfirmed = "scrape failure announcement not confirmed, " +
		"a caller waiting for this page may learn nothing until it stops waiting"
	msgReceiptCarried        = "intake receipt carried onto the page feed"
	msgReceiptSubjectUnknown = "intake receipt carried on no page feed, " +
		"a caller waiting for this page learns nothing until it stops waiting"
	msgReceiptNotCarried = "intake receipt not carried onto the page feed, " +
		"a caller waiting for this page learns nothing until it stops waiting"
)

type ScrapeOutcomeFeedLog struct{}

func (ScrapeOutcomeFeedLog) ScrapeFailureAnnounced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
) {
	slog.DebugContext(ctx, msgFailureAnnounced,
		slog.String("pageUrl", pageURL.String()),
		slog.String("subject", subject),
	)
}

func (ScrapeOutcomeFeedLog) ScrapeFailureEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgFailureUnreadable,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeOutcomeFeedLog) ScrapeFailurePublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
	cause error,
) {
	slog.WarnContext(ctx, msgFailureNotAnnounced,
		slog.String("pageUrl", pageURL.String()),
		slog.String("subject", subject),
		slog.Any("error", cause),
	)
}

func (ScrapeOutcomeFeedLog) ScrapeFailureConfirmationFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
	cause error,
) {
	slog.WarnContext(ctx, msgFailureUnconfirmed,
		slog.String("pageUrl", pageURL.String()),
		slog.String("subject", subject),
		slog.Any("error", cause),
	)
}

func (ScrapeOutcomeFeedLog) IntakeReceiptCarried(
	ctx context.Context,
	receiptSubject string,
	outcomeSubject string,
) {
	slog.DebugContext(ctx, msgReceiptCarried,
		slog.String("receiptSubject", receiptSubject),
		slog.String("outcomeSubject", outcomeSubject),
	)
}

func (ScrapeOutcomeFeedLog) IntakeReceiptSubjectUnreadable(
	ctx context.Context,
	receiptSubject string,
	cause error,
) {
	slog.WarnContext(ctx, msgReceiptSubjectUnknown,
		slog.String("receiptSubject", receiptSubject),
		slog.Any("error", cause),
	)
}

func (ScrapeOutcomeFeedLog) IntakeReceiptNotCarried(
	ctx context.Context,
	receiptSubject string,
	outcomeSubject string,
	cause error,
) {
	slog.WarnContext(ctx, msgReceiptNotCarried,
		slog.String("receiptSubject", receiptSubject),
		slog.String("outcomeSubject", outcomeSubject),
		slog.Any("error", cause),
	)
}
