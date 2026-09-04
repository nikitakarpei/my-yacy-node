package pageintake

import (
	"context"

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
	NoMarkdownDerived(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
	)
	MarkdownNotStored(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	MarkdownStored(
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

func (observers IntakeProgressObservers) NoMarkdownDerived(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.NoMarkdownDerived(ctx, pageURL)
	}
}

func (observers IntakeProgressObservers) MarkdownNotStored(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.MarkdownNotStored(ctx, pageURL, cause)
	}
}

func (observers IntakeProgressObservers) MarkdownStored(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	for _, observer := range observers {
		observer.MarkdownStored(ctx, pageURL)
	}
}
