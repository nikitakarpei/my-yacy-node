package pageread

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageReadObserver interface {
	PageAnswered(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchOutcome FetchOutcome,
	)
	MarkdownRecallFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		cause error,
	)
	FetchOutcomeNotHeard(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchWait time.Duration,
	)
}

type PageReadObservers []PageReadObserver

func (observers PageReadObservers) PageAnswered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchOutcome FetchOutcome,
) {
	for _, observer := range observers {
		observer.PageAnswered(ctx, pageURL, fetchOutcome)
	}
}

func (observers PageReadObservers) MarkdownRecallFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.MarkdownRecallFailed(ctx, pageURL, cause)
	}
}

func (observers PageReadObservers) FetchOutcomeNotHeard(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchWait time.Duration,
) {
	for _, observer := range observers {
		observer.FetchOutcomeNotHeard(ctx, pageURL, fetchWait)
	}
}
