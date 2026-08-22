// Package fetchtiming times a fetch and reports its elapsed duration, then returns the fetch.
package fetchtiming

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/clock"
)

type FetchProgress interface {
	FetchCompleted(elapsed time.Duration)
}

type Fetch struct {
	observer FetchProgress
	clock    clock.Clock
	fetcher  pagefetch.Fetcher
}

func New(observer FetchProgress, clock clock.Clock, fetcher pagefetch.Fetcher) *Fetch {
	return &Fetch{observer: observer, clock: clock, fetcher: fetcher}
}

func (f *Fetch) Fetch(
	ctx context.Context,
	rawURL string,
	knownVersion pagefetch.PageVersion,
) (pagefetch.FetchOutcome, error) {
	start := f.clock.Now()
	outcome, err := f.fetcher.Fetch(ctx, rawURL, knownVersion)
	if err != nil {
		return pagefetch.FetchOutcome{}, err
	}
	f.observer.FetchCompleted(f.clock.Now().Sub(start))
	return outcome, nil
}
