package networksearch

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type pagePrefetcher struct {
	documentsOrdering DocumentsOrdering
	pageReadingRun    PageReadingRun
	pagesReadPerQuery int
	pagesReadPerSite  int
	latestFindings    chan queryfindings.Findings
	pagesReadAhead    []pagereading.PageToRead
	stopAsked         chan struct{}
	stopped           chan struct{}
}

func (n Network) prefetcherFor(pageReadingRun PageReadingRun) *pagePrefetcher {
	return &pagePrefetcher{
		documentsOrdering: n.documentsOrdering,
		pageReadingRun:    pageReadingRun,
		pagesReadPerQuery: n.pagesReadPerQuery,
		pagesReadPerSite:  n.pagesReadPerSite,
		latestFindings:    make(chan queryfindings.Findings, 1),
		stopAsked:         make(chan struct{}),
		stopped:           make(chan struct{}),
	}
}

func (prefetcher *pagePrefetcher) Start(ctx context.Context) {
	go prefetcher.readAheadUntilStopped(ctx)
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
	pagesToRead := pagesToReadAmong(
		prefetcher.documentsOrdering.OrderedDocumentsOf(findings),
		prefetcher.pagesReadPerQuery,
		prefetcher.pagesReadPerSite,
	)
	prefetcher.pageReadingRun.Abandon(pagesLeftOutOf(pagesToRead, prefetcher.pagesReadAhead))
	prefetcher.pageReadingRun.ReadAhead(ctx, pagesToRead)
	prefetcher.pagesReadAhead = pagesToRead
}

func pagesLeftOutOf(
	pagesToRead []pagereading.PageToRead,
	pagesReadAhead []pagereading.PageToRead,
) []pagereading.PageToRead {
	documentsToRead := make(map[yacymodel.URLHash]struct{}, len(pagesToRead))
	for _, pageToRead := range pagesToRead {
		documentsToRead[pageToRead.Document] = struct{}{}
	}
	pagesLeftOut := make([]pagereading.PageToRead, 0, len(pagesReadAhead))
	for _, pageReadAhead := range pagesReadAhead {
		if _, toRead := documentsToRead[pageReadAhead.Document]; !toRead {
			pagesLeftOut = append(pagesLeftOut, pageReadAhead)
		}
	}

	return pagesLeftOut
}
