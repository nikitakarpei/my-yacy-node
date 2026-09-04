package pagevisit

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
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

type PageFetcher struct {
	inner    pagefetch.Fetcher
	clock    Clock
	observer PageFetchObserver
}

func NewPageFetcher(
	inner pagefetch.Fetcher,
	clock Clock,
	observer PageFetchObserver,
) *PageFetcher {
	return &PageFetcher{inner: inner, clock: clock, observer: observer}
}

func (f *PageFetcher) Fetch(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	fetchStarted := f.clock.Now()
	fetchOutcome, err := f.inner.Fetch(ctx, pageURL, knownVersion)
	fetchDuration := f.clock.Now().Sub(fetchStarted)
	if err != nil {
		f.observer.PageFetchFailed(ctx, pageURL, fetchDuration, err)
		return fetchOutcome, err
	}
	f.observeFetchOutcome(ctx, pageURL, fetchDuration, fetchOutcome)
	return fetchOutcome, nil
}

func (f *PageFetcher) observeFetchOutcome(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
	fetchOutcome pagefetch.FetchOutcome,
) {
	switch fetchOutcome.Status {
	case pagefetch.FetchSucceeded:
		f.observer.PageFetchSucceeded(ctx, pageURL, fetchDuration)
	case pagefetch.FetchNotModified:
		f.observer.PageFetchNotModified(ctx, pageURL, fetchDuration)
	case pagefetch.FetchAccessRefused:
		f.observer.PageFetchAccessRefused(ctx, pageURL, fetchDuration)
	case pagefetch.FetchDeferred:
		f.observer.PageFetchDeferred(ctx, pageURL, fetchDuration, fetchOutcome.DeferFor)
	case pagefetch.FetchRejected:
		f.observer.PageFetchRejected(ctx, pageURL, fetchDuration)
	case pagefetch.FetchLandedURLInvalid:
		f.observer.PageFetchLandedURLInvalid(
			ctx, pageURL, fetchDuration, fetchOutcome.FailureCause,
		)
	case pagefetch.FetchOversized:
		f.observer.PageFetchRefusedOversizedPage(ctx, pageURL, fetchDuration)
	case pagefetch.FetchFailed:
		f.observer.PageFetchFailed(ctx, pageURL, fetchDuration, fetchOutcome.FailureCause)
	}
}
