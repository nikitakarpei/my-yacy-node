package pagereading

import (
	"context"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type startedPages struct {
	mutex                  sync.Mutex
	pagePerDocument        map[yacymodel.URLHash]*startedPage
	documentsWanted        map[yacymodel.URLHash]struct{}
	amountOfPagesAbandoned int
	finished               bool
}

func noStartedPages() *startedPages {
	return &startedPages{
		pagePerDocument: map[yacymodel.URLHash]*startedPage{},
		documentsWanted: map[yacymodel.URLHash]struct{}{},
	}
}

func (pages *startedPages) addAhead(
	ctx context.Context,
	pagesToRead []PageToRead,
) ([]*startedPage, bool) {
	pages.mutex.Lock()
	defer pages.mutex.Unlock()
	if pages.finished {
		return nil, false
	}

	return pages.pagesNewlyStarted(ctx, pagesToRead), true
}

func (pages *startedPages) abandon(pagesToAbandon []PageToRead) {
	pages.mutex.Lock()
	defer pages.mutex.Unlock()
	for _, pageToAbandon := range pagesToAbandon {
		page, started := pages.pagePerDocument[pageToAbandon.Document]
		_, wanted := pages.documentsWanted[pageToAbandon.Document]
		if !started || wanted || page.isSettled() {
			continue
		}
		page.abandon()
		delete(pages.pagePerDocument, pageToAbandon.Document)
		pages.amountOfPagesAbandoned++
	}
}

func (pages *startedPages) addWanted(
	ctx context.Context,
	pagesWanted []PageToRead,
) ([]*startedPage, bool) {
	pages.mutex.Lock()
	defer pages.mutex.Unlock()
	if pages.finished {
		return nil, false
	}
	for _, pageWanted := range pagesWanted {
		pages.documentsWanted[pageWanted.Document] = struct{}{}
	}

	return pages.pagesNewlyStarted(ctx, pagesWanted), true
}

func (pages *startedPages) pagesNewlyStarted(
	ctx context.Context,
	pagesToRead []PageToRead,
) []*startedPage {
	newPages := make([]*startedPage, 0, len(pagesToRead))
	for _, pageToRead := range pagesToRead {
		if _, started := pages.pagePerDocument[pageToRead.Document]; started {
			continue
		}
		page := startedPageOf(ctx, pageToRead)
		pages.pagePerDocument[pageToRead.Document] = page
		newPages = append(newPages, page)
	}

	return newPages
}

func (pages *startedPages) want(pagesWanted []PageToRead) <-chan pageReadResult {
	pages.mutex.Lock()
	defer pages.mutex.Unlock()
	settlingResults := make(chan pageReadResult, len(pagesWanted))
	for _, pageWanted := range pagesWanted {
		page := pages.pagePerDocument[pageWanted.Document]
		go func() { settlingResults <- page.pageReadResultOnceSettled() }()
	}

	return settlingResults
}

func (pages *startedPages) finish() FinishedPageReadingRun {
	pages.mutex.Lock()
	defer pages.mutex.Unlock()
	pages.finished = true
	amountOfPagesUnwanted := 0
	for startedDocument := range pages.pagePerDocument {
		if _, wanted := pages.documentsWanted[startedDocument]; !wanted {
			amountOfPagesUnwanted++
		}
	}

	return FinishedPageReadingRun{
		AmountOfPagesUnwanted:  amountOfPagesUnwanted,
		AmountOfPagesAbandoned: pages.amountOfPagesAbandoned,
	}
}
