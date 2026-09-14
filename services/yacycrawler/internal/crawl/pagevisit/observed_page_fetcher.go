package pagevisit

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
)

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		knownVersion pagefetch.PageVersion,
	) pagefetch.FetchOutcome
}

type ObservedPageFetcher struct {
	inner    pagefetch.Fetcher
	clock    Clock
	observer PageFetchObserver
}

func NewObservedPageFetcher(
	inner pagefetch.Fetcher,
	clock Clock,
	observer PageFetchObserver,
) *ObservedPageFetcher {
	return &ObservedPageFetcher{inner: inner, clock: clock, observer: observer}
}

func (f *ObservedPageFetcher) Fetch(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	knownVersion pagefetch.PageVersion,
) pagefetch.FetchOutcome {
	fetchStarted := f.clock.Now()
	fetchOutcome, err := f.inner.Fetch(ctx, pageURL, knownVersion)
	fetchDuration := f.clock.Now().Sub(fetchStarted)
	if err != nil {
		f.observeUnfinishedFetch(ctx, pageURL, fetchDuration, err)
		return pagefetch.FetchOutcome{Status: pagefetch.FetchFailed}
	}
	f.observeFetchOutcome(ctx, pageURL, fetchDuration, fetchOutcome)
	return fetchOutcome
}

func (f *ObservedPageFetcher) observeUnfinishedFetch(
	ctx context.Context,
	pageURL canonicalurl.CanonicalURL,
	fetchDuration time.Duration,
	cause error,
) {
	if ctx.Err() != nil {
		f.observer.PageFetchCanceled(ctx, pageURL, fetchDuration)
		return
	}
	f.observer.PageFetchFailed(ctx, pageURL, fetchDuration, cause)
}

func (f *ObservedPageFetcher) observeFetchOutcome(
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
	case pagefetch.FetchRedirected:
		f.observer.PageFetchRedirected(
			ctx, pageURL, fetchOutcome.RedirectTarget, fetchDuration,
		)
	case pagefetch.FetchRedirectTargetInvalid:
		f.observer.PageFetchRedirectTargetInvalid(
			ctx, pageURL, fetchDuration, fetchOutcome.FailureCause,
		)
	case pagefetch.FetchOversized:
		f.observer.PageFetchRefusedOversizedPage(ctx, pageURL, fetchDuration)
	case pagefetch.FetchFailed:
		f.observer.PageFetchFailed(ctx, pageURL, fetchDuration, fetchOutcome.FailureCause)
	}
}
