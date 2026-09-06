package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageFetchSucceeded             = "page fetch succeeded"
	msgPageFetchNotModified           = "page fetch not modified"
	msgPageFetchAccessRefused         = "page fetch access refused"
	msgPageFetchDeferred              = "page fetch deferred"
	msgPageFetchRejected              = "page fetch rejected"
	msgPageFetchRedirected            = "page fetch redirected"
	msgPageFetchRedirectTargetInvalid = "page fetch redirect target invalid"
	msgPageFetchRefusedOversizedPage  = "page fetch refused oversized page"
	msgPageFetchFailed                = "page fetch failed"
	msgPageFetchCanceled              = "page fetch canceled before it finished"
)

type PageFetchLog struct{}

func (PageFetchLog) PageFetchSucceeded(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchSucceeded, slog.String("url", pageURL.String()))
}

func (PageFetchLog) PageFetchNotModified(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchNotModified, slog.String("url", pageURL.String()))
}

func (PageFetchLog) PageFetchAccessRefused(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchAccessRefused, slog.String("url", pageURL.String()))
}

func (PageFetchLog) PageFetchDeferred(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
	deferFor time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchDeferred,
		slog.String("url", pageURL.String()),
		slog.Duration("deferFor", deferFor),
	)
}

func (PageFetchLog) PageFetchRejected(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchRejected, slog.String("url", pageURL.String()))
}

func (PageFetchLog) PageFetchRedirected(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	redirectTarget canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchRedirected,
		slog.String("url", pageURL.String()),
		slog.String("redirectTarget", redirectTarget.String()),
	)
}

func (PageFetchLog) PageFetchRedirectTargetInvalid(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgPageFetchRedirectTargetInvalid,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageFetchLog) PageFetchRefusedOversizedPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchRefusedOversizedPage, slog.String("url", pageURL.String()))
}

func (PageFetchLog) PageFetchFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgPageFetchFailed,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageFetchLog) PageFetchCanceled(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchCanceled, slog.String("url", pageURL.String()))
}
