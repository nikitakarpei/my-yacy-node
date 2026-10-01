package pagereading

import (
	"context"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Run struct {
	pageReadResultOf func(ctx context.Context, pageToRead PageToRead) pageReadResult
	mutex            sync.Mutex
	startedPages     map[yacymodel.URLHash]*startedPage
}

func newRun(
	pageReadResultOf func(ctx context.Context, pageToRead PageToRead) pageReadResult,
) *Run {
	return &Run{
		pageReadResultOf: pageReadResultOf,
		startedPages:     map[yacymodel.URLHash]*startedPage{},
	}
}

func (run *Run) Read(ctx context.Context, pagesToRead []PageToRead) {
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

func (run *Run) pageReadResultsAmong(pagesWanted []PageToRead) <-chan pageReadResult {
	run.mutex.Lock()
	defer run.mutex.Unlock()
	pageReadResults := make(chan pageReadResult, len(pagesWanted))
	for _, pageWanted := range pagesWanted {
		page := run.startedPages[pageWanted.Document]
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
