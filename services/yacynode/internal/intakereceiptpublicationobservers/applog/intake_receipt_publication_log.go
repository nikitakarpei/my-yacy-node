// Package applog writes a log line for each intake receipt the corpus sends back to the
// caller waiting on a page, and for each receipt that never leaves. It is the only place
// that decides how a fact reads and at which level it is written.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgReceiptSent       = "intake receipt sent"
	msgReceiptUnreadable = "intake receipt cannot be written, " +
		"a caller waiting for this page learns nothing until it stops waiting"
	msgReceiptNotPublished = "intake receipt not published, " +
		"a caller waiting for this page learns nothing until it stops waiting"
	msgReceiptUnconfirmed = "intake receipt publication not confirmed, " +
		"a caller waiting for this page may learn nothing until it stops waiting"
)

type IntakeReceiptPublicationLog struct{}

func (IntakeReceiptPublicationLog) IntakeReceiptSent(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
) {
	slog.DebugContext(ctx, msgReceiptSent,
		slog.String("pageUrl", pageURL.String()),
		slog.String("subject", subject),
	)
}

func (IntakeReceiptPublicationLog) IntakeReceiptEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgReceiptUnreadable,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (IntakeReceiptPublicationLog) IntakeReceiptPublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
	cause error,
) {
	slog.WarnContext(ctx, msgReceiptNotPublished,
		slog.String("pageUrl", pageURL.String()),
		slog.String("subject", subject),
		slog.Any("error", cause),
	)
}

func (IntakeReceiptPublicationLog) IntakeReceiptConfirmationFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
	cause error,
) {
	slog.WarnContext(ctx, msgReceiptUnconfirmed,
		slog.String("pageUrl", pageURL.String()),
		slog.String("subject", subject),
		slog.Any("error", cause),
	)
}
