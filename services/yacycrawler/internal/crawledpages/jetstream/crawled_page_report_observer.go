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

type CrawledPageReportObserver interface {
	CrawledPageReported(
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
	CrawledPageReportingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		indexing PageIndexing,
		cause error,
	)
}

type CrawledPageReportObservers []CrawledPageReportObserver

func (observers CrawledPageReportObservers) CrawledPageReported(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing PageIndexing,
) {
	for _, observer := range observers {
		observer.CrawledPageReported(ctx, pageURL, indexing)
	}
}

func (observers CrawledPageReportObservers) CrawledPageEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing PageIndexing,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawledPageEncodingFailed(ctx, pageURL, indexing, cause)
	}
}

func (observers CrawledPageReportObservers) CrawledPageReportingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	indexing PageIndexing,
	cause error,
) {
	for _, observer := range observers {
		observer.CrawledPageReportingFailed(ctx, pageURL, indexing, cause)
	}
}
