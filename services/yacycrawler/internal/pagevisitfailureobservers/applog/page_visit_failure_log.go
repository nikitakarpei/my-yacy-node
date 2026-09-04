package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const msgPageHTMLUnreadable = "page html unreadable, page visit left for another attempt"

type PageVisitFailureLog struct{}

func (PageVisitFailureLog) PageHTMLUnreadable(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(
		ctx,
		msgPageHTMLUnreadable,
		slog.String("url", pageURL.String()),
		slog.Any("error", cause),
	)
}
