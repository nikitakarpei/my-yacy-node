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
	msgPageAnswered              = "page call answered"
	msgMarkdownRecallFailed      = "markdown corpus recall failed, the caller gets an error"
	msgScrapeOutcomeListenFailed = "scrape outcome listener not opened, " +
		"the caller waits for nothing and reads what the corpus already holds"
	msgScrapeRequestFailed = "scrape request not published, " +
		"the caller reads what the corpus already holds"
	msgFetchOutcomeNotHeard = "what became of the page fetch was not heard, " +
		"the caller reads what the corpus already holds"
)

type PageReadProgressLog struct{}

func (PageReadProgressLog) PageAnswered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchOutcome pageread.FetchOutcome,
) {
	slog.DebugContext(ctx, msgPageAnswered,
		slog.String("pageUrl", pageURL.String()),
		slog.String("fetchOutcome", string(fetchOutcome)),
	)
}

func (PageReadProgressLog) MarkdownRecallFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgMarkdownRecallFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageReadProgressLog) ScrapeOutcomeListenFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeOutcomeListenFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageReadProgressLog) ScrapeRequestFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeRequestFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (PageReadProgressLog) FetchOutcomeNotHeard(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchWait time.Duration,
	cause error,
) {
	slog.WarnContext(ctx, msgFetchOutcomeNotHeard,
		slog.String("pageUrl", pageURL.String()),
		slog.Duration("fetchWait", fetchWait),
		slog.Any("error", cause),
	)
}
