package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgRecrawlRecordFailed = "recrawl record failed, next visit may be redundant"

type RecrawlRecordLog struct{}

func (RecrawlRecordLog) RecrawlRecordFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgRecrawlRecordFailed,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}
