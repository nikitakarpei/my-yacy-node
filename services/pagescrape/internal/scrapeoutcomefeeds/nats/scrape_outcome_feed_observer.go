package nats

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeOutcomeFeedObserver interface {
	ScrapeFailureAnnounced(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		subject string,
	)
	ScrapeFailureEncodingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	ScrapeFailurePublishingFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		subject string,
		cause error,
	)
	ScrapeFailureConfirmationFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		subject string,
		cause error,
	)
	IntakeReceiptCarried(
		ctx context.Context,
		receiptSubject string,
		outcomeSubject string,
	)
	IntakeReceiptSubjectUnreadable(
		ctx context.Context,
		receiptSubject string,
		cause error,
	)
	IntakeReceiptNotCarried(
		ctx context.Context,
		receiptSubject string,
		outcomeSubject string,
		cause error,
	)
}

type ScrapeOutcomeFeedObservers []ScrapeOutcomeFeedObserver

func (observers ScrapeOutcomeFeedObservers) ScrapeFailureAnnounced(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
) {
	for _, observer := range observers {
		observer.ScrapeFailureAnnounced(ctx, pageURL, subject)
	}
}

func (observers ScrapeOutcomeFeedObservers) ScrapeFailureEncodingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeFailureEncodingFailed(ctx, pageURL, cause)
	}
}

func (observers ScrapeOutcomeFeedObservers) ScrapeFailurePublishingFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeFailurePublishingFailed(ctx, pageURL, subject, cause)
	}
}

func (observers ScrapeOutcomeFeedObservers) ScrapeFailureConfirmationFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	subject string,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeFailureConfirmationFailed(ctx, pageURL, subject, cause)
	}
}

func (observers ScrapeOutcomeFeedObservers) IntakeReceiptCarried(
	ctx context.Context,
	receiptSubject string,
	outcomeSubject string,
) {
	for _, observer := range observers {
		observer.IntakeReceiptCarried(ctx, receiptSubject, outcomeSubject)
	}
}

func (observers ScrapeOutcomeFeedObservers) IntakeReceiptSubjectUnreadable(
	ctx context.Context,
	receiptSubject string,
	cause error,
) {
	for _, observer := range observers {
		observer.IntakeReceiptSubjectUnreadable(ctx, receiptSubject, cause)
	}
}

func (observers ScrapeOutcomeFeedObservers) IntakeReceiptNotCarried(
	ctx context.Context,
	receiptSubject string,
	outcomeSubject string,
	cause error,
) {
	for _, observer := range observers {
		observer.IntakeReceiptNotCarried(ctx, receiptSubject, outcomeSubject, cause)
	}
}
