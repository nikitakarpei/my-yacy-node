package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgPageFetchSucceeded            = "page fetch succeeded"
	msgPageFetchNotModified          = "page fetch not modified"
	msgPageFetchAccessRefused        = "page fetch access refused"
	msgPageFetchDeferred             = "page fetch deferred"
	msgPageFetchRejected             = "page fetch rejected"
	msgPageFetchLandedURLInvalid     = "page fetch landed url invalid"
	msgPageFetchRefusedOversizedPage = "page fetch refused oversized page"
	msgPageFetchFailed               = "page fetch failed"
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

func (PageFetchLog) PageFetchLandedURLInvalid(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	_ time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgPageFetchLandedURLInvalid,
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
