// Package applog writes each page intake fact to the application log.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageRWIReceived        = "rwi of page received"
	msgPageRWIInvalid         = "rwi of page invalid, nothing stored"
	msgPostingsRootMismatched = "rwi of page has a postings root other than the root of this node, " +
		"intake halted until the partition exponents of the pagerwi service and the node match"
	msgPageAdmitted        = "rwi of page admitted"
	msgPageAdmissionBusy   = "rwi of page admission deferred because storage is busy"
	msgPageAdmissionFailed = "rwi of page admission failed"
	msgPageIndexed         = "rwi of page indexed"
)

type PageIntakeLog struct{}

func (PageIntakeLog) PageRWIInvalid(ctx context.Context) {
	slog.WarnContext(ctx, msgPageRWIInvalid)
}

func (PageIntakeLog) PageRWIReceived(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgPageRWIReceived,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (PageIntakeLog) PostingsRootMismatched(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.ErrorContext(ctx, msgPostingsRootMismatched,
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
