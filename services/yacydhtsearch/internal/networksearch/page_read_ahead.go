package networksearch

import (
	"context"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

type pageReadAhead struct {
	pagesToReadFrom func(queryfindings.Findings) []pagereading.PageToRead
	pageReadingRun  PageReadingRun
	newerFindings   chan struct{}
	readAheadsEnded chan struct{}
	mutex           sync.Mutex
	findings        queryfindings.Findings
}

func newPageReadAhead(
	pagesToReadFrom func(queryfindings.Findings) []pagereading.PageToRead,
	pageReadingRun PageReadingRun,
) *pageReadAhead {
	return &pageReadAhead{
		pagesToReadFrom: pagesToReadFrom,
		pageReadingRun:  pageReadingRun,
		newerFindings:   make(chan struct{}, 1),
		readAheadsEnded: make(chan struct{}),
	}
}

func (r *pageReadAhead) start(ctx context.Context) {
	go r.readAheadAsFindingsGrow(ctx)
}

func (r *pageReadAhead) readAheadAsFindingsGrow(ctx context.Context) {
	defer close(r.readAheadsEnded)
	for range r.newerFindings {
		r.readAheadOf(ctx, r.latestFindings())
	}
}

func (r *pageReadAhead) latestFindings() queryfindings.Findings {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.findings
}

func (r *pageReadAhead) readAheadOf(ctx context.Context, findings queryfindings.Findings) {
	r.pageReadingRun.ReadAhead(ctx, r.pagesToReadFrom(findings))
}

func (r *pageReadAhead) findingsGrew(findings queryfindings.Findings) {
	r.mutex.Lock()
	r.findings = findings
	r.mutex.Unlock()
	select {
	case r.newerFindings <- struct{}{}:
	default:
	}
}

func (r *pageReadAhead) stop() {
	close(r.newerFindings)
	<-r.readAheadsEnded
}
