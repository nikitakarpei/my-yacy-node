package jetstream

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeRequestPublicationObserver interface {
	ScrapeRequestPublished(ctx context.Context, pageURL canonicalurl.CanonicalURL)
	ScrapeRequestEncodingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	ScrapeRequestPublishingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
}

type ScrapeRequestPublicationObservers []ScrapeRequestPublicationObserver

func (observers ScrapeRequestPublicationObservers) ScrapeRequestPublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.ScrapeRequestPublished(ctx, pageURL)
	}
}

func (observers ScrapeRequestPublicationObservers) ScrapeRequestEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeRequestEncodingFailed(ctx, pageURL, cause)
	}
}

func (observers ScrapeRequestPublicationObservers) ScrapeRequestPublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeRequestPublishingFailed(ctx, pageURL, cause)
	}
}
