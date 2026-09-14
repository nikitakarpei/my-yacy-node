// Package applog writes a log line for each fact the service learns while it answers a call
// for one page, so an operator can read the whole life of one call in the service log. It is
// the only place that decides how a fact reads and at which level it is written.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/webresearchmcp/internal/pageread"
)

const (
	msgPageAnswered         = "page call answered"
	msgMarkdownRecallFailed = "markdown corpus recall failed, the caller gets an error"
	msgFetchOutcomeNotHeard = "what became of the page fetch was not heard, " +
		"the caller reads what the corpus already holds"
)

type PageReadLog struct{}

func (PageReadLog) PageAnswered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchOutcome pageread.FetchOutcome,
) {
	slog.DebugContext(ctx, msgPageAnswered,
		slog.String("pageUrl", pageURL.String()),
		slog.String("fetchOutcome", string(fetchOutcome)),
	)
}

func (PageReadLog) MarkdownRecallFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgMarkdownRecallFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageReadLog) FetchOutcomeNotHeard(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchWait time.Duration,
) {
	slog.WarnContext(ctx, msgFetchOutcomeNotHeard,
		slog.String("pageUrl", pageURL.String()),
		slog.Duration("fetchWait", fetchWait),
	)
}
