package pagereading

import (
	"context"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Run struct {
	reading Reading
	//nolint:containedctx // the run reads each page it starts under the context it started in
	readingCtx   context.Context
	stopReading  context.CancelFunc
	queryWords   []yacymodel.Hash
	mutex        sync.Mutex
	startedPages map[yacymodel.URLHash]*startedPage
}

func (run *Run) Read(pagesToRead []PageToRead) {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	for _, pageToRead := range pagesToRead {
		if _, started := run.startedPages[pageToRead.Document]; started {
			continue
		}
		page := &startedPage{settled: make(chan struct{})}
		run.startedPages[pageToRead.Document] = page
		go func() {
			page.settleWith(
				run.reading.pageReadResultOf(run.readingCtx, run.queryWords, pageToRead),
			)
		}()
	}
}

func (run *Run) ReadPagesAmong(ctx context.Context, pagesWanted []PageToRead) ReadPages {
	startedAt := time.Now()
	run.Read(pagesWanted)
	pageReadResults := run.reading.pageReadResultsWithinTheBudget(
		ctx, run.stopReading, run.settledPagesAmong(pagesWanted), pagesWanted,
	)
	run.reading.reportPageReading(
		ctx,
		performedPageReadingFrom(
			pageReadResults,
			run.amountOfPagesUnwanted(pagesWanted),
			time.Since(startedAt),
		),
	)

	return readPagesFrom(pageReadResults)
}

func (run *Run) settledPagesAmong(pagesWanted []PageToRead) <-chan settledPage {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	settledPages := make(chan settledPage, len(pagesWanted))
	for place, pageWanted := range pagesWanted {
		page := run.startedPages[pageWanted.Document]
		go func() {
			settledPages <- settledPage{
				place:          place,
				pageReadResult: page.pageReadResultOnceSettled(),
			}
		}()
	}

	return settledPages
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
