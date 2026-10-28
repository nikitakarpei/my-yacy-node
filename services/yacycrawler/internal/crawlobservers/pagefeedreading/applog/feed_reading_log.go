package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgFeedUnreadable = "feed unreadable, following the links read before the break"

type FeedReadingLog struct{}

func (FeedReadingLog) FeedUnreadable(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgFeedUnreadable,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}
