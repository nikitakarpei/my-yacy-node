// Package applog writes a log line for each fact the service learns while it waits for the
// end of one scrape. It is the only place that decides how a fact reads and at which level
// it is written.
package applog

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgScrapeOutcomeSubscriptionFailed = "scrape outcome feed not subscribed to, " +
		"the caller reads what the corpus already holds"
	msgScrapeOutcomeListenerUnconfirmed = "scrape outcome listener not confirmed, " +
		"the caller reads what the corpus already holds"
	msgScrapeOutcomeMessageMalformed = "scrape outcome message not read, " +
		"the caller keeps waiting for another one"
)

type ScrapeOutcomeLog struct{}

func (ScrapeOutcomeLog) ScrapeOutcomeSubscriptionFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeOutcomeSubscriptionFailed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeOutcomeLog) ScrapeOutcomeListenerConfirmationFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeOutcomeListenerUnconfirmed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeOutcomeLog) ScrapeOutcomeMessageMalformed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgScrapeOutcomeMessageMalformed,
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}
