package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgPageVisitNotRecorded = "page visit not recorded, the next crawl fetches the page again"

type PageVisitRecordLog struct{}

func (PageVisitRecordLog) PageVisitNotRecorded(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgPageVisitNotRecorded,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}
