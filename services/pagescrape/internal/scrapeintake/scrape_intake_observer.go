package scrapeintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type ScrapeIntakeObserver interface {
	ScrapeRequestInvalid(
		ctx context.Context,
		message string,
		cause error,
	)
	ScrapeRequestReceived(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
	)
	OriginFetchFailed(
		ctx context.Context,
		fetchURL canonicalurl.CanonicalURL,
		cause error,
	)
	PageOffered(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		landedURL canonicalurl.CanonicalURL,
	)
	PageNotOffered(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	ScrapeDeferred(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		deferFor time.Duration,
	)
	ScrapeScheduleFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	ScrapeFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		reason pagescrapecontract.ScrapeFailureReason,
	)
}

type ScrapeIntakeObservers []ScrapeIntakeObserver

func (observers ScrapeIntakeObservers) ScrapeRequestInvalid(
	ctx context.Context,
	message string,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeRequestInvalid(ctx, message, cause)
	}
}

func (observers ScrapeIntakeObservers) ScrapeRequestReceived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.ScrapeRequestReceived(ctx, pageURL)
	}
}

func (observers ScrapeIntakeObservers) OriginFetchFailed(
	ctx context.Context,
	fetchURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.OriginFetchFailed(ctx, fetchURL, cause)
	}
}

func (observers ScrapeIntakeObservers) PageOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	landedURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageOffered(ctx, pageURL, landedURL)
	}
}

func (observers ScrapeIntakeObservers) PageNotOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.PageNotOffered(ctx, pageURL, cause)
	}
}

func (observers ScrapeIntakeObservers) ScrapeDeferred(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	deferFor time.Duration,
) {
	for _, observer := range observers {
		observer.ScrapeDeferred(ctx, pageURL, deferFor)
	}
}

func (observers ScrapeIntakeObservers) ScrapeScheduleFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeScheduleFailed(ctx, pageURL, cause)
	}
}

func (observers ScrapeIntakeObservers) ScrapeFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	reason pagescrapecontract.ScrapeFailureReason,
) {
	for _, observer := range observers {
		observer.ScrapeFailed(ctx, pageURL, reason)
	}
}
