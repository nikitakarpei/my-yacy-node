package pageintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageIntakeObserver interface {
	PageOffered(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
	)
	NoDocumentExtracted(
		ctx context.Context,
		landedURL canonicalurl.CanonicalURL,
		cause error,
	)
	NoReadableTextDerived(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
	)
	IndexObserved(
		ctx context.Context,
		elapsed time.Duration,
	)
	IndexFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	PageIndexed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
	)
}

type PageIntakeObservers []PageIntakeObserver

func (observers PageIntakeObservers) PageOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageOffered(ctx, pageURL)
	}
}

func (observers PageIntakeObservers) NoDocumentExtracted(
	ctx context.Context,
	landedURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.NoDocumentExtracted(ctx, landedURL, cause)
	}
}

func (observers PageIntakeObservers) NoReadableTextDerived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NoReadableTextDerived(ctx, pageURL)
	}
}

func (observers PageIntakeObservers) IndexObserved(
	ctx context.Context,
	elapsed time.Duration,
) {
	for _, observer := range observers {
		observer.IndexObserved(ctx, elapsed)
	}
}

func (observers PageIntakeObservers) IndexFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.IndexFailed(ctx, pageURL, cause)
	}
}

func (observers PageIntakeObservers) PageIndexed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageIndexed(ctx, pageURL)
	}
}
