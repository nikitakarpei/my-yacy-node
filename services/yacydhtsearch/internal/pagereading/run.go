package pagereading

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Run struct {
	pageReader   pageReader
	queryWords   []yacymodel.Hash
	deadlines    pageReadDeadlines
	observer     PageReadingObserver
	startedPages *startedPages
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
		startedPages: noStartedPages(),
	}
}

func (run *Run) ReadAhead(ctx context.Context, pagesToRead []PageToRead) {
	pagesToStart, open := run.startedPages.addAhead(ctx, pagesToRead)
	if !open {
		run.observer.PagesReadAheadAfterTheFinish(ctx, len(pagesToRead))

		return
	}
	run.startReading(pagesToStart)
}

func (run *Run) startReading(pagesToStart []*startedPage) {
	for _, page := range pagesToStart {
		go func() {
			page.settleWith(run.pageReader.read(page.readContext, run.queryWords, page.pageToRead))
		}()
	}
}

func (run *Run) Read(ctx context.Context, pagesWanted []PageToRead) PagesRead {
	startedAt := time.Now()
	pagesToStart, open := run.startedPages.addWanted(ctx, pagesWanted)
	if !open {
		run.observer.PagesReadAfterTheFinish(ctx, len(pagesWanted))

		return PagesRead{}
	}
	run.startReading(pagesToStart)
	deadline := run.deadlines.deadlineFor(len(pagesWanted))
	defer deadline.stop()
	settlingResults := run.startedPages.want(pagesWanted)
	pageReadResults := awaitPageReadResults(settlingResults, deadline, pagesWanted)
	run.observer.PageReadingPerformed(
		ctx, performedPageReadingFrom(pageReadResults, time.Since(startedAt)),
	)

	return pagesReadFrom(pageReadResults)
}

func awaitPageReadResults(
	settlingResults <-chan pageReadResult,
	deadline *pageReadDeadline,
	pagesWanted []PageToRead,
) pageReadResults {
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

func (run *Run) Finish(ctx context.Context) {
	run.observer.PageReadingRunFinished(
		ctx, run.startedPages.finish(),
	)
}
