// Package applog writes each page intake fact to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageOffered              = "offered page received"
	msgOfferedPageInvalid       = "offered page invalid, nothing stored"
	msgDocumentExtractionFailed = "offered page document extraction failed, nothing stored"
	msgNoIndexDerived           = "offered page derives no index, nothing stored"
	msgPageAdmitted             = "offered page admitted"
	msgPageAdmissionBusy        = "offered page admission deferred because storage is busy"
	msgPageAdmissionFailed      = "offered page admission failed"
	msgPageIndexed              = "offered page indexed"
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

func (PageIntakeLog) PageAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	slog.DebugContext(ctx, msgPageAdmitted,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", amountOfPostings),
	)
}

func (PageIntakeLog) PageAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	slog.WarnContext(ctx, msgPageAdmissionBusy,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", amountOfPostings),
	)
}

func (PageIntakeLog) PageAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	amountOfPostings int,
	cause error,
) {
	slog.WarnContext(ctx, msgPageAdmissionFailed,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", amountOfPostings),
		slog.Any("error", cause),
	)
}

func (PageIntakeLog) PageIndexed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgPageIndexed,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}
