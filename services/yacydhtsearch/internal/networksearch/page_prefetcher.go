package networksearch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

type pagePrefetcher struct {
	documentsOrdering DocumentsOrdering
	pageReadingRun    PageReadingRun
	pagesReadPerQuery int
	pagesReadPerSite  int
	latestFindings    chan queryfindings.Findings
	stopAsked         chan struct{}
	stopped           chan struct{}
}

func (n Network) prefetcherStartedFor(
	ctx context.Context,
	pageReadingRun PageReadingRun,
) *pagePrefetcher {
	prefetcher := &pagePrefetcher{
		documentsOrdering: n.documentsOrdering,
		pageReadingRun:    pageReadingRun,
		pagesReadPerQuery: n.pagesReadPerQuery,
		pagesReadPerSite:  n.pagesReadPerSite,
		latestFindings:    make(chan queryfindings.Findings, 1),
		stopAsked:         make(chan struct{}),
		stopped:           make(chan struct{}),
	}
	go prefetcher.readAheadUntilStopped(ctx)

	return prefetcher
}

func (prefetcher *pagePrefetcher) FindingsGrew(findings queryfindings.Findings) {
	select {
	case <-prefetcher.latestFindings:
	default:
	}
	prefetcher.latestFindings <- findings
}

func (prefetcher *pagePrefetcher) Stop() {
	close(prefetcher.stopAsked)
	<-prefetcher.stopped
}

func (prefetcher *pagePrefetcher) readAheadUntilStopped(ctx context.Context) {
	defer close(prefetcher.stopped)
	for {
		select {
		case findings := <-prefetcher.latestFindings:
			prefetcher.readAheadThePagesOf(ctx, findings)
		case <-prefetcher.stopAsked:
			return
		}
	}
}

func (prefetcher *pagePrefetcher) readAheadThePagesOf(
	ctx context.Context,
	findings queryfindings.Findings,
) {
	prefetcher.pageReadingRun.ReadAhead(ctx, pagesToReadAmong(
		prefetcher.documentsOrdering.OrderedDocumentsOf(findings),
		prefetcher.pagesReadPerQuery,
		prefetcher.pagesReadPerSite,
	))
}
