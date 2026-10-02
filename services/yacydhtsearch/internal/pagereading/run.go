package pagereading

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Run struct {
	pageReader               pageReader
	queryWords               []yacymodel.Hash
	deadlines                pageReadDeadlines
	observer                 PageReadingObserver
	mutex                    sync.Mutex
	startedPages             map[yacymodel.URLHash]*startedPage
	finished                 bool
	pagesWanted              []PageToRead
	pageReadResultsWaitedFor pageReadResults
	timeSpentWaiting         time.Duration
}

func newRun(
	reader pageReader,
	queryWords []yacymodel.Hash,
	deadlines pageReadDeadlines,
	observer PageReadingObserver,
) *Run {
	return &Run{
		pageReader:   reader,
		queryWords:   queryWords,
		deadlines:    deadlines,
		observer:     observer,
		startedPages: map[yacymodel.URLHash]*startedPage{},
	}
}

func (run *Run) StartReading(ctx context.Context, pagesToRead []PageToRead) {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	if run.finished {
		return
	}
	for _, pageToRead := range pagesToRead {
		if _, started := run.startedPages[pageToRead.Document]; started {
			continue
		}
		page := &startedPage{settled: make(chan struct{})}
		run.startedPages[pageToRead.Document] = page
		go func() { page.settleWith(run.pageReader.read(ctx, run.queryWords, pageToRead)) }()
	}
}

func (run *Run) PagesReadAmong(_ context.Context, pagesWanted []PageToRead) PagesRead {
	startedAt := time.Now()
	pageReadResults := run.pageReadResultsFrom(run.pageReadResultsAmong(pagesWanted), pagesWanted)
	run.recordWait(pagesWanted, pageReadResults, time.Since(startedAt))

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

func (run *Run) pageReadResultsFrom(
	settlingResults <-chan pageReadResult,
	pagesWanted []PageToRead,
) pageReadResults {
	deadline := run.deadlines.deadlineFor(len(pagesWanted))
	defer deadline.stop()
	pageReadResults := make(pageReadResults, 0, len(pagesWanted))
	for len(pageReadResults) < len(pagesWanted) {
		select {
		case pageReadResult := <-settlingResults:
			pageReadResults = append(pageReadResults, pageReadResult)
			deadline.pageSettled()
		case <-deadline.graceEnded():
			return pageReadResults.withUnsettledPagesCutOff(pagesWanted)
		case <-deadline.budgetEnded():
			return pageReadResults.withUnsettledPagesOutOfBudget(pagesWanted)
		}
	}

	return pageReadResults
}

func (run *Run) recordWait(
	pagesWanted []PageToRead,
	pageReadResults pageReadResults,
	timeSpentWaiting time.Duration,
) {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	run.pagesWanted = pagesWanted
	run.pageReadResultsWaitedFor = pageReadResults
	run.timeSpentWaiting = timeSpentWaiting
}

func (run *Run) Finish(ctx context.Context) {
	run.stopStarting()
	run.reportPageReading(ctx, run.performedPageReading())
}

func (run *Run) stopStarting() {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	run.finished = true
}

func (run *Run) performedPageReading() PerformedPageReading {
	run.mutex.Lock()
	defer run.mutex.Unlock()

	return performedPageReadingFrom(
		run.pageReadResultsWaitedFor,
		run.amountOfPagesUnwanted(),
		run.timeSpentWaiting,
	)
}

func (run *Run) amountOfPagesUnwanted() int {
	documentsWanted := make(map[yacymodel.URLHash]struct{}, len(run.pagesWanted))
	for _, pageWanted := range run.pagesWanted {
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
