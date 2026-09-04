package pagevisit

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

type PageFetchObserver interface {
	PageFetchSucceeded(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration time.Duration,
	)
	PageFetchNotModified(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration time.Duration,
	)
	PageFetchAccessRefused(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration time.Duration,
	)
	PageFetchDeferred(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration, deferFor time.Duration,
	)
	PageFetchRejected(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration time.Duration,
	)
	PageFetchLandedURLInvalid(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration time.Duration,
		cause error,
	)
	PageFetchRefusedOversizedPage(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration time.Duration,
	)
	PageFetchFailed(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration time.Duration,
		cause error,
	)
	PageFetchCanceled(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		fetchDuration time.Duration,
	)
}

type PageFetchObservers []PageFetchObserver

func (observers PageFetchObservers) PageFetchSucceeded(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
) {
	for _, observer := range observers {
		observer.PageFetchSucceeded(ctx, pageURL, fetchDuration)
	}
}

func (observers PageFetchObservers) PageFetchNotModified(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
) {
	for _, observer := range observers {
		observer.PageFetchNotModified(ctx, pageURL, fetchDuration)
	}
}

func (observers PageFetchObservers) PageFetchAccessRefused(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
) {
	for _, observer := range observers {
		observer.PageFetchAccessRefused(ctx, pageURL, fetchDuration)
	}
}

func (observers PageFetchObservers) PageFetchDeferred(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
	deferFor time.Duration,
) {
	for _, observer := range observers {
		observer.PageFetchDeferred(ctx, pageURL, fetchDuration, deferFor)
	}
}

func (observers PageFetchObservers) PageFetchRejected(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
) {
	for _, observer := range observers {
		observer.PageFetchRejected(ctx, pageURL, fetchDuration)
	}
}

func (observers PageFetchObservers) PageFetchLandedURLInvalid(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
	cause error,
) {
	for _, observer := range observers {
		observer.PageFetchLandedURLInvalid(ctx, pageURL, fetchDuration, cause)
	}
}

func (observers PageFetchObservers) PageFetchRefusedOversizedPage(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
) {
	for _, observer := range observers {
		observer.PageFetchRefusedOversizedPage(ctx, pageURL, fetchDuration)
	}
}

func (observers PageFetchObservers) PageFetchFailed(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
	cause error,
) {
	for _, observer := range observers {
		observer.PageFetchFailed(ctx, pageURL, fetchDuration, cause)
	}
}

func (observers PageFetchObservers) PageFetchCanceled(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
) {
	for _, observer := range observers {
		observer.PageFetchCanceled(ctx, pageURL, fetchDuration)
	}
}
