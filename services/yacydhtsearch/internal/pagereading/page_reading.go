// Package pagereading reads the page of each document one query puts first,
// all at once inside one read budget, and gives back the text of each document
// it could read. It takes the readable text of the page, and the whole text
// when the page holds no readable article. A page it cannot fetch, read, or
// finish inside the budget gives back nothing for its document.
package pagereading

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageToRead struct {
	Document yacymodel.URLHash
	Address  string
}

type FormatDerivations interface {
	BodyIn(
		ctx context.Context,
		format documentextraction.Format,
		document documentextraction.Document,
		pageURL canonicalurl.CanonicalURL,
	) ([]byte, bool)
}

type Reading struct {
	pageFetch            pagefetch.Fetcher
	formatDerivations    FormatDerivations
	pageReadBudget       time.Duration
	snippetLengthCeiling int
	observer             PageReadingObserver
}

func New(
	pageFetch pagefetch.Fetcher,
	formatDerivations FormatDerivations,
	pageReadBudget time.Duration,
	snippetLengthCeiling int,
	observer PageReadingObserver,
) Reading {
	return Reading{
		pageFetch:            pageFetch,
		formatDerivations:    formatDerivations,
		pageReadBudget:       pageReadBudget,
		snippetLengthCeiling: snippetLengthCeiling,
		observer:             observer,
	}
}

func (r Reading) DocumentTextPerDocument(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	pagesToRead []PageToRead,
) map[yacymodel.URLHash]documenttext.DocumentText {
	startedAt := time.Now()
	budgetedCtx, stopPageReadBudget := context.WithTimeout(ctx, r.pageReadBudget)
	defer stopPageReadBudget()

	readPages := r.readPagesOf(budgetedCtx, queryWords, pagesToRead)
	r.observer.PageReadingPerformed(
		ctx,
		performedPageReadingFrom(readPages, time.Since(startedAt)),
	)

	return documentTextPerDocumentOf(readPages)
}

func (r Reading) readPagesOf(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	pagesToRead []PageToRead,
) []readPage {
	readPages := make([]readPage, len(pagesToRead))
	var pagesBeingRead sync.WaitGroup
	for place, pageToRead := range pagesToRead {
		pagesBeingRead.Add(1)
		go func() {
			defer pagesBeingRead.Done()
			readPages[place] = r.readPageOf(ctx, queryWords, pageToRead)
		}()
	}
	pagesBeingRead.Wait()

	return readPages
}

func (r Reading) readPageOf(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	pageToRead PageToRead,
) readPage {
	pageURL, err := canonicalurl.CanonicalURLOf(pageToRead.Address)
	if err != nil {
		return readPage{document: pageToRead.Document, outcome: pageUnreachable}
	}
	fetchStartedAt := time.Now()
	fetched, err := r.pageFetch.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	timeSpentFetching := time.Since(fetchStartedAt)
	if outcome := outcomeOfAFetch(fetched, err, ctx.Err()); outcome != pageRead {
		return readPage{
			document:          pageToRead.Document,
			outcome:           outcome,
			timeSpentFetching: timeSpentFetching,
		}
	}

	readingStartedAt := time.Now()
	read := r.readPageFromTheBody(ctx, queryWords, pageToRead, fetched.Page, pageURL)
	read.timeSpentFetching = timeSpentFetching
	read.timeSpentReading = time.Since(readingStartedAt)

	return read
}

func outcomeOfAFetch(
	fetched pagefetch.FetchOutcome,
	fetchFailure error,
	budgetFailure error,
) readOutcome {
	if fetchFailure != nil {
		return outcomeOfAFetchFailure(fetchFailure, budgetFailure)
	}
	switch fetched.Status {
	case pagefetch.FetchSucceeded:
		return pageRead
	case pagefetch.FetchDeadlinePassed:
		return pageOutOfBudget
	default:
		return pageRefused
	}
}

func outcomeOfAFetchFailure(fetchFailure error, budgetFailure error) readOutcome {
	if budgetFailure != nil || errors.Is(fetchFailure, context.DeadlineExceeded) {
		return pageOutOfBudget
	}

	return pageUnreachable
}

func (r Reading) readPageFromTheBody(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	pageToRead PageToRead,
	page pagefetch.FetchedPage,
	pageURL canonicalurl.CanonicalURL,
) readPage {
	document, err := documentextraction.DocumentFrom(
		ctx, page.Body, page.ContentType, pageURL,
	)
	if err != nil {
		return readPage{
			document: pageToRead.Document,
			outcome:  outcomeOfAnExtractionFailure(err),
		}
	}

	return r.readPageFromTheDocument(ctx, queryWords, pageToRead, document, pageURL)
}

func outcomeOfAnExtractionFailure(extractionFailure error) readOutcome {
	if errors.Is(extractionFailure, documentextraction.ErrUnsupportedMediaType) {
		return pageOfAnUnsupportedKind
	}

	return pageUnreadable
}

func (r Reading) readPageFromTheDocument(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	pageToRead PageToRead,
	document documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) readPage {
	text, derived := r.textOfTheDocument(ctx, document, pageURL)
	if !derived {
		return readPage{document: pageToRead.Document, outcome: pageUnreadable}
	}

	return readPage{
		document: pageToRead.Document,
		outcome:  pageRead,
		text:     documenttext.DocumentTextFrom(string(text), queryWords, r.snippetLengthCeiling),
	}
}

func (r Reading) textOfTheDocument(
	ctx context.Context,
	document documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) ([]byte, bool) {
	readableText, readableTextDerived := r.formatDerivations.BodyIn(
		ctx, documentextraction.FormatReadableText, document, pageURL,
	)
	if readableTextDerived && len(bytes.TrimSpace(readableText)) > 0 {
		return readableText, true
	}

	return r.formatDerivations.BodyIn(
		ctx, documentextraction.FormatFullText, document, pageURL,
	)
}

func documentTextPerDocumentOf(
	readPages []readPage,
) map[yacymodel.URLHash]documenttext.DocumentText {
	documentTextPerDocument := make(
		map[yacymodel.URLHash]documenttext.DocumentText, len(readPages),
	)
	for _, readPage := range readPages {
		if readPage.outcome != pageRead {
			continue
		}
		documentTextPerDocument[readPage.document] = readPage.text
	}

	return documentTextPerDocument
}
