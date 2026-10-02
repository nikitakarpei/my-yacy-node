package pagereading

import (
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type startedPages struct {
	mutex           sync.Mutex
	pagePerDocument map[yacymodel.URLHash]*startedPage
	documentsWanted map[yacymodel.URLHash]struct{}
	finished        bool
}

func noStartedPages() *startedPages {
	return &startedPages{
		pagePerDocument: map[yacymodel.URLHash]*startedPage{},
		documentsWanted: map[yacymodel.URLHash]struct{}{},
	}
}

func (pages *startedPages) add(pagesToRead []PageToRead) ([]*startedPage, bool) {
	pages.mutex.Lock()
	defer pages.mutex.Unlock()
	if pages.finished {
		return nil, false
	}
	newPages := make([]*startedPage, 0, len(pagesToRead))
	for _, pageToRead := range pagesToRead {
		if _, started := pages.pagePerDocument[pageToRead.Document]; started {
			continue
		}
		page := &startedPage{pageToRead: pageToRead, settled: make(chan struct{})}
		pages.pagePerDocument[pageToRead.Document] = page
		newPages = append(newPages, page)
	}

	return newPages, true
}

func (pages *startedPages) want(pagesWanted []PageToRead) <-chan pageReadResult {
	pages.mutex.Lock()
	defer pages.mutex.Unlock()
	settlingResults := make(chan pageReadResult, len(pagesWanted))
	for _, pageWanted := range pagesWanted {
		pages.documentsWanted[pageWanted.Document] = struct{}{}
		page := pages.pagePerDocument[pageWanted.Document]
		go func() { settlingResults <- page.pageReadResultOnceSettled() }()
	}

	return settlingResults
}

func (pages *startedPages) finish() int {
	pages.mutex.Lock()
	defer pages.mutex.Unlock()
	pages.finished = true
	amountOfPagesUnwanted := 0
	for startedDocument := range pages.pagePerDocument {
		if _, wanted := pages.documentsWanted[startedDocument]; !wanted {
			amountOfPagesUnwanted++
		}
	}

	return amountOfPagesUnwanted
}
