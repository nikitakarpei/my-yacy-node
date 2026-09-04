package jetstream

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageIndexing string

const (
	PageAllowsIndexing  PageIndexing = "allowed"
	PageRefusesIndexing PageIndexing = "refused"
)

type CrawledPagePublicationObserver interface {
	CrawledPagePublished(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		indexing PageIndexing,
	)
	CrawledPageEncodingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		indexing PageIndexing,
		cause error,
	)
	CrawledPagePublishingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		indexing PageIndexing,
		cause error,
	)
}

type CrawledPagePublicationObservers []CrawledPagePublicationObserver

func (observers CrawledPagePublicationObservers) CrawledPagePublished(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing PageIndexing,
) {
	for _, observer := range observers {
		observer.CrawledPagePublished(ctx, pageURL, indexing)
	}
}

func (observers CrawledPagePublicationObservers) CrawledPageEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing PageIndexing,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawledPageEncodingFailed(ctx, pageURL, indexing, cause)
	}
}

func (observers CrawledPagePublicationObservers) CrawledPagePublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing PageIndexing,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawledPagePublishingFailed(ctx, pageURL, indexing, cause)
	}
}
