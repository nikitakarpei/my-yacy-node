// Package applog writes each page intake fact to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageOffered                = "offered page received"
	msgOfferedPageInvalid         = "offered page invalid, nothing stored"
	msgDocumentExtractionFailed   = "offered page document extraction failed, nothing stored"
	msgNoIndexDerived             = "offered page derives no index, nothing stored"
	msgURLMetadataAdmitted        = "offered page url metadata admitted"
	msgURLMetadataAdmissionBusy   = "offered page url metadata admission deferred because storage is busy"
	msgURLMetadataAdmissionFailed = "offered page url metadata admission failed"
	msgPostingsAdmitted           = "offered page postings admitted"
	msgPostingsAdmissionBusy      = "offered page postings admission deferred because storage is busy"
	msgPostingsAdmissionFailed    = "offered page postings admission failed"
	msgPageIndexed                = "offered page indexed"
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

func (PageIntakeLog) URLMetadataAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgURLMetadataAdmitted,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (PageIntakeLog) URLMetadataAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.WarnContext(ctx, msgURLMetadataAdmissionBusy,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (PageIntakeLog) URLMetadataAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	attributes := []any{
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	}
	if cause != nil {
		attributes = append(attributes, slog.Any("error", cause))
	}
	slog.WarnContext(ctx, msgURLMetadataAdmissionFailed, attributes...)
}

func (PageIntakeLog) PostingsAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	slog.DebugContext(ctx, msgPostingsAdmitted,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", postings),
	)
}

func (PageIntakeLog) PostingsAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	slog.WarnContext(ctx, msgPostingsAdmissionBusy,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", postings),
	)
}

func (PageIntakeLog) PostingsAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
	cause error,
) {
	slog.WarnContext(ctx, msgPostingsAdmissionFailed,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", postings),
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
