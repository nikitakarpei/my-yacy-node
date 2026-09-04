package pageread

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageReadProgress interface {
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

type PageReadProgressObservers []PageReadProgress

func (observers PageReadProgressObservers) PageAnswered(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchOutcome FetchOutcome,
) {
	for _, observer := range observers {
		observer.PageAnswered(ctx, pageURL, fetchOutcome)
	}
}

func (observers PageReadProgressObservers) MarkdownRecallFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	cause error,
) {
	for _, observer := range observers {
		observer.MarkdownRecallFailed(ctx, pageURL, cause)
	}
}

func (observers PageReadProgressObservers) FetchOutcomeNotHeard(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchWait time.Duration,
) {
	for _, observer := range observers {
		observer.FetchOutcomeNotHeard(ctx, pageURL, fetchWait)
	}
}
