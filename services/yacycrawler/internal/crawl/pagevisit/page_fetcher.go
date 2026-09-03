package pagevisit

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type PageFetchObserver interface {
	PageFetchCompleted(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		status pagefetch.FetchStatus,
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

func (observers PageFetchObservers) PageFetchCompleted(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	status pagefetch.FetchStatus,
	fetchDuration time.Duration,
) {
	for _, observer := range observers {
		observer.PageFetchCompleted(ctx, pageURL, status, fetchDuration)
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
	f.observer.PageFetchCompleted(ctx, pageURL, fetchOutcome.Status, fetchDuration)
	return fetchOutcome, nil
}
