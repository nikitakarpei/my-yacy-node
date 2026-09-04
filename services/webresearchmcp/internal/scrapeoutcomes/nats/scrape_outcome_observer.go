package nats

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type ScrapeOutcomeObserver interface {
	ScrapeOutcomeSubscriptionFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	ScrapeOutcomeListenerConfirmationFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	ScrapeOutcomeMessageMalformed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
}

type ScrapeOutcomeObservers []ScrapeOutcomeObserver

func (observers ScrapeOutcomeObservers) ScrapeOutcomeSubscriptionFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeOutcomeSubscriptionFailed(ctx, pageURL, cause)
	}
}

func (observers ScrapeOutcomeObservers) ScrapeOutcomeListenerConfirmationFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeOutcomeListenerConfirmationFailed(ctx, pageURL, cause)
	}
}

func (observers ScrapeOutcomeObservers) ScrapeOutcomeMessageMalformed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.ScrapeOutcomeMessageMalformed(ctx, pageURL, cause)
	}
}
