package pageintake

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type IntakeProgress interface {
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

type IntakeProgressObservers []IntakeProgress

func (observers IntakeProgressObservers) PageOffered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageOffered(ctx, pageURL)
	}
}

func (observers IntakeProgressObservers) NoDocumentExtracted(
	ctx context.Context,
	landedURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.NoDocumentExtracted(ctx, landedURL, cause)
	}
}

func (observers IntakeProgressObservers) NoReadableTextDerived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NoReadableTextDerived(ctx, pageURL)
	}
}

func (observers IntakeProgressObservers) IndexObserved(
	ctx context.Context,
	elapsed time.Duration,
) {
	for _, observer := range observers {
		observer.IndexObserved(ctx, elapsed)
	}
}

func (observers IntakeProgressObservers) IndexFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.IndexFailed(ctx, pageURL, cause)
	}
}

func (observers IntakeProgressObservers) PageIndexed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.PageIndexed(ctx, pageURL)
	}
}
