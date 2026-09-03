package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

const (
	msgPageFetchCompleted = "page fetch completed"
	msgPageFetchFailed    = "page fetch failed"
)

type PageFetchLog struct{}

func (PageFetchLog) PageFetchCompleted(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	status pagefetch.FetchStatus,
	_ time.Duration,
) {
	slog.DebugContext(ctx, msgPageFetchCompleted,
		slog.String("url", pageURL.String()),
		slog.Int("status", int(status)),
	)
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
