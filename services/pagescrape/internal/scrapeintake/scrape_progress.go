package scrapeintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type ScrapeProgress interface {
	ScrapeRequestInvalid(
		ctx context.Context,
		message string,
		cause error,
	)
	ScrapeRequestReceived(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
	)
	OriginReadFailed(
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

type ScrapeProgressObservers []ScrapeProgress

func (observers ScrapeProgressObservers) ScrapeRequestInvalid(
	ctx context.Context,
	message string,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeRequestInvalid(ctx, message, cause)
	}
}

func (observers ScrapeProgressObservers) ScrapeRequestReceived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.ScrapeRequestReceived(ctx, pageURL)
	}
}

func (observers ScrapeProgressObservers) OriginReadFailed(
	ctx context.Context,
	fetchURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.OriginReadFailed(ctx, fetchURL, cause)
	}
}

func (observers ScrapeProgressObservers) PageOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	landedURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageOffered(ctx, pageURL, landedURL)
	}
}

func (observers ScrapeProgressObservers) PageNotOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.PageNotOffered(ctx, pageURL, cause)
	}
}

func (observers ScrapeProgressObservers) ScrapeDeferred(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	deferFor time.Duration,
) {
	for _, observer := range observers {
		observer.ScrapeDeferred(ctx, pageURL, deferFor)
	}
}

func (observers ScrapeProgressObservers) ScrapeScheduleFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeScheduleFailed(ctx, pageURL, cause)
	}
}

func (observers ScrapeProgressObservers) ScrapeFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	reason pagescrapecontract.ScrapeFailureReason,
) {
	for _, observer := range observers {
		observer.ScrapeFailed(ctx, pageURL, reason)
	}
}
