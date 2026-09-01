// Package applog writes scrape request progress to the application log.
package applog

import (
	"context"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

const (
	msgOriginFetchFailed          = "scrape request fetch failed"
	msgOriginFetchDeferred        = "scrape request fetch deferred by the origin"
	msgNothingToScrape            = "scrape request fetch holds no content to scrape"
	msgDocumentExtractionFailed   = "scrape request document extraction failed, nothing stored"
	msgNoIndexDerived             = "scrape request derives no index, nothing stored"
	msgURLMetadataAdmitted        = "scrape request url metadata admitted"
	msgURLMetadataAdmissionBusy   = "scrape request url metadata admission deferred because storage is busy"
	msgURLMetadataAdmissionFailed = "scrape request url metadata admission failed"
	msgPostingsAdmitted           = "scrape request postings admitted"
	msgPostingsAdmissionBusy      = "scrape request postings admission deferred because storage is busy"
	msgPostingsAdmissionFailed    = "scrape request postings admission failed"
	msgScrapeRequestCompleted     = "scrape request completed"
)

type ScrapeProgressLog struct{}

func (ScrapeProgressLog) ScrapeRequestInvalid(context.Context) {}

func (ScrapeProgressLog) OriginFetchFailed(
	ctx context.Context,
	messageIdentity string,
	fetchURL canonicalurl.CanonicalURL,
	cause error,
) {
	attributes := []any{
		slog.String("message", messageIdentity),
		slog.String("fetchUrl", fetchURL.String()),
	}
	if cause != nil {
		attributes = append(attributes, slog.Any("error", cause))
	}
	slog.WarnContext(ctx, msgOriginFetchFailed, attributes...)
}

func (ScrapeProgressLog) OriginFetchDeferred(
	ctx context.Context,
	messageIdentity string,
	fetchURL canonicalurl.CanonicalURL,
	deferFor time.Duration,
) {
	slog.DebugContext(ctx, msgOriginFetchDeferred,
		slog.String("message", messageIdentity),
		slog.String("fetchUrl", fetchURL.String()),
		slog.Duration("deferFor", deferFor),
	)
}

func (ScrapeProgressLog) NothingToScrape(
	ctx context.Context,
	messageIdentity string,
	fetchURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgNothingToScrape,
		slog.String("message", messageIdentity),
		slog.String("fetchUrl", fetchURL.String()),
	)
}

func (ScrapeProgressLog) DocumentExtractionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	slog.WarnContext(ctx, msgDocumentExtractionFailed,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) NoIndexDerived(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgNoIndexDerived,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (ScrapeProgressLog) URLMetadataAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgURLMetadataAdmitted,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (ScrapeProgressLog) URLMetadataAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.WarnContext(ctx, msgURLMetadataAdmissionBusy,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}

func (ScrapeProgressLog) URLMetadataAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	attributes := []any{
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	}
	if cause != nil {
		attributes = append(attributes, slog.Any("error", cause))
	}
	slog.WarnContext(ctx, msgURLMetadataAdmissionFailed, attributes...)
}

func (ScrapeProgressLog) PostingsAdmitted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	slog.DebugContext(ctx, msgPostingsAdmitted,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", postings),
	)
}

func (ScrapeProgressLog) PostingsAdmissionBusy(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
) {
	slog.WarnContext(ctx, msgPostingsAdmissionBusy,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", postings),
	)
}

func (ScrapeProgressLog) PostingsAdmissionFailed(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
	postings int,
	cause error,
) {
	slog.WarnContext(ctx, msgPostingsAdmissionFailed,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
		slog.Int("postings", postings),
		slog.Any("error", cause),
	)
}

func (ScrapeProgressLog) ScrapeRequestCompleted(
	ctx context.Context,
	messageIdentity string,
	pageURL canonicalurl.CanonicalURL,
) {
	slog.DebugContext(ctx, msgScrapeRequestCompleted,
		slog.String("message", messageIdentity),
		slog.String("pageUrl", pageURL.String()),
	)
}
