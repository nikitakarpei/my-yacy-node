package pagereading

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Run struct {
	pageReadResultOf func(ctx context.Context, pageToRead PageToRead) pageReadResult
	waiting          pageReadWaiting
	observer         PageReadingObserver
	mutex            sync.Mutex
	startedPages     map[yacymodel.URLHash]*startedPage
}

func newRun(
	pageReadResultOf func(ctx context.Context, pageToRead PageToRead) pageReadResult,
	waiting pageReadWaiting,
	observer PageReadingObserver,
) *Run {
	return &Run{
		pageReadResultOf: pageReadResultOf,
		waiting:          waiting,
		observer:         observer,
		startedPages:     map[yacymodel.URLHash]*startedPage{},
	}
}

func (run *Run) StartReading(ctx context.Context, pagesToRead []PageToRead) {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	for _, pageToRead := range pagesToRead {
		if _, started := run.startedPages[pageToRead.Document]; started {
			continue
		}
		page := &startedPage{settled: make(chan struct{})}
		run.startedPages[pageToRead.Document] = page
		go func() { page.settleWith(run.pageReadResultOf(ctx, pageToRead)) }()
	}
}

func (run *Run) PagesReadAmong(ctx context.Context, pagesWanted []PageToRead) PagesRead {
	startedAt := time.Now()
	pageReadResults := run.waiting.pageReadResultsFrom(
		ctx, run.pageReadResultsAmong(pagesWanted), pagesWanted,
	)
	run.reportPageReading(
		ctx,
		performedPageReadingFrom(
			pageReadResults,
			run.amountOfPagesUnwanted(pagesWanted),
			time.Since(startedAt),
		),
	)

	return pagesReadFrom(pageReadResults)
}

func (run *Run) pageReadResultsAmong(pagesWanted []PageToRead) <-chan pageReadResult {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	pageReadResults := make(chan pageReadResult, len(pagesWanted))
	for _, pageWanted := range pagesWanted {
		page, started := run.startedPages[pageWanted.Document]
		if !started {
			pageReadResults <- pageReadResult{
				document: pageWanted.Document,
				outcome:  pageWasOutOfBudget,
			}

			continue
		}
		go func() { pageReadResults <- page.pageReadResultOnceSettled() }()
	}

	return pageReadResults
}

func (run *Run) amountOfPagesUnwanted(pagesWanted []PageToRead) int {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	documentsWanted := make(map[yacymodel.URLHash]struct{}, len(pagesWanted))
	for _, pageWanted := range pagesWanted {
		documentsWanted[pageWanted.Document] = struct{}{}
	}
	amountOfPagesUnwanted := 0
	for startedDocument := range run.startedPages {
		if _, wanted := documentsWanted[startedDocument]; !wanted {
			amountOfPagesUnwanted++
		}
	}

	return amountOfPagesUnwanted
}

func (run *Run) reportPageReading(ctx context.Context, pageReading PerformedPageReading) {
	run.observer.PageReadingPerformed(ctx, pageReading)
}
