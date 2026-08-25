package markdownintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeProgress interface {
	ScrapeRequestReceived(ctx context.Context)
	OriginFetchFailed(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
		cause error,
	)
	OriginFetchDeferred(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
		deferFor time.Duration,
	)
	NothingToScrape(ctx context.Context, requestedURL canonicalurl.CanonicalURL)
	DocumentExtractionFailed(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	NoMarkdownDerived(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
		pageURL canonicalurl.CanonicalURL,
	)
	MarkdownCorpusWriteFailed(
		ctx context.Context,
		markdownURL canonicalurl.CanonicalURL,
		cause error,
	)
	RedirectionRecordWriteFailed(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
		markdownURL canonicalurl.CanonicalURL,
		cause error,
	)
	MarkdownStored(
		ctx context.Context,
		requestedURL canonicalurl.CanonicalURL,
		markdownURL canonicalurl.CanonicalURL,
	)
}

type ScrapeProgressObservers []ScrapeProgress

func (observers ScrapeProgressObservers) ScrapeRequestReceived(ctx context.Context) {
	for _, observer := range observers {
		observer.ScrapeRequestReceived(ctx)
	}
}

func (observers ScrapeProgressObservers) OriginFetchFailed(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.OriginFetchFailed(ctx, requestedURL, cause)
	}
}

func (observers ScrapeProgressObservers) OriginFetchDeferred(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	deferFor time.Duration,
) {
	for _, observer := range observers {
		observer.OriginFetchDeferred(ctx, requestedURL, deferFor)
	}
}

func (observers ScrapeProgressObservers) NothingToScrape(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NothingToScrape(ctx, requestedURL)
	}
}

func (observers ScrapeProgressObservers) DocumentExtractionFailed(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.DocumentExtractionFailed(ctx, requestedURL, pageURL, cause)
	}
}

func (observers ScrapeProgressObservers) NoMarkdownDerived(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NoMarkdownDerived(ctx, requestedURL, pageURL)
	}
}

func (observers ScrapeProgressObservers) MarkdownCorpusWriteFailed(
	ctx context.Context,
	markdownURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.MarkdownCorpusWriteFailed(ctx, markdownURL, cause)
	}
}

func (observers ScrapeProgressObservers) RedirectionRecordWriteFailed(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	markdownURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.RedirectionRecordWriteFailed(ctx, requestedURL, markdownURL, cause)
	}
}

func (observers ScrapeProgressObservers) MarkdownStored(
	ctx context.Context,
	requestedURL canonicalurl.CanonicalURL,
	markdownURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.MarkdownStored(ctx, requestedURL, markdownURL)
	}
}
