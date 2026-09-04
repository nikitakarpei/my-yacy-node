package nats

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type IntakeReceiptPublicationObserver interface {
	IntakeReceiptSent(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		subject string,
	)
	IntakeReceiptEncodingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	IntakeReceiptPublishingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		subject string,
		cause error,
	)
	IntakeReceiptConfirmationFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		subject string,
		cause error,
	)
}

type IntakeReceiptPublicationObservers []IntakeReceiptPublicationObserver

func (observers IntakeReceiptPublicationObservers) IntakeReceiptSent(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
) {
	for _, observer := range observers {
		observer.IntakeReceiptSent(ctx, pageURL, subject)
	}
}

func (observers IntakeReceiptPublicationObservers) IntakeReceiptEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.IntakeReceiptEncodingFailed(ctx, pageURL, cause)
	}
}

func (observers IntakeReceiptPublicationObservers) IntakeReceiptPublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
	cause error,
) {
	for _, observer := range observers {
		observer.IntakeReceiptPublishingFailed(ctx, pageURL, subject, cause)
	}
}

func (observers IntakeReceiptPublicationObservers) IntakeReceiptConfirmationFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
	cause error,
) {
	for _, observer := range observers {
		observer.IntakeReceiptConfirmationFailed(ctx, pageURL, subject, cause)
	}
}
