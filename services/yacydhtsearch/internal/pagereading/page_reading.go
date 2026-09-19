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
		extractedDocument documentextraction.Document,
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
		return readPage{document: pageToRead.Document, outcome: pageWasUnreachable}
	}
	fetchStartedAt := time.Now()
	fetched, err := r.pageFetch.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	timeSpentFetching := time.Since(fetchStartedAt)
	if err != nil {
		return readPage{
			document:          pageToRead.Document,
			outcome:           readOutcomeFromAFetchFailure(err, ctx.Err()),
			timeSpentFetching: timeSpentFetching,
		}
	}
	if fetched.Status != pagefetch.FetchSucceeded {
		return readPage{
			document:          pageToRead.Document,
			outcome:           readOutcomeFromAFetchStatus(fetched.Status),
			timeSpentFetching: timeSpentFetching,
		}
	}

	readingStartedAt := time.Now()
	text, outcome := r.documentTextFromTheFetchedPage(ctx, queryWords, fetched.Page, pageURL)

	return readPage{
		document:          pageToRead.Document,
		outcome:           outcome,
		text:              text,
		timeSpentFetching: timeSpentFetching,
		timeSpentReading:  time.Since(readingStartedAt),
	}
}

func readOutcomeFromAFetchFailure(fetchFailure error, budgetFailure error) readOutcome {
	if budgetFailure != nil || errors.Is(fetchFailure, context.DeadlineExceeded) {
		return pageWasOutOfBudget
	}

	return pageWasUnreachable
}

func readOutcomeFromAFetchStatus(status pagefetch.FetchStatus) readOutcome {
	if status == pagefetch.FetchDeadlinePassed {
		return pageWasOutOfBudget
	}

	return pageWasRefused
}

func (r Reading) documentTextFromTheFetchedPage(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	fetchedPage pagefetch.FetchedPage,
	pageURL canonicalurl.CanonicalURL,
) (documenttext.DocumentText, readOutcome) {
	extractedDocument, err := documentextraction.DocumentFrom(
		ctx, fetchedPage.Body, fetchedPage.ContentType, pageURL,
	)
	if err != nil {
		return documenttext.DocumentText{}, readOutcomeFromAnExtractionFailure(err)
	}
	text, derived := r.textOfTheExtractedDocument(ctx, extractedDocument, pageURL)
	if !derived {
		return documenttext.DocumentText{}, pageWasUnreadable
	}

	return documenttext.DocumentTextFrom(
		extractedDocument.Title, string(text), queryWords, r.snippetLengthCeiling,
	), pageWasRead
}

func readOutcomeFromAnExtractionFailure(extractionFailure error) readOutcome {
	if errors.Is(extractionFailure, documentextraction.ErrUnsupportedMediaType) {
		return pageWasOfAnUnsupportedKind
	}

	return pageWasUnreadable
}

func (r Reading) textOfTheExtractedDocument(
	ctx context.Context,
	extractedDocument documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) ([]byte, bool) {
	readableText, readableTextDerived := r.formatDerivations.BodyIn(
		ctx, documentextraction.FormatReadableText, extractedDocument, pageURL,
	)
	if readableTextDerived && len(bytes.TrimSpace(readableText)) > 0 {
		return readableText, true
	}

	return r.formatDerivations.BodyIn(
		ctx, documentextraction.FormatFullText, extractedDocument, pageURL,
	)
}

func documentTextPerDocumentOf(
	readPages []readPage,
) map[yacymodel.URLHash]documenttext.DocumentText {
	documentTextPerDocument := make(
		map[yacymodel.URLHash]documenttext.DocumentText, len(readPages),
	)
	for _, readPage := range readPages {
		if readPage.outcome != pageWasRead {
			continue
		}
		documentTextPerDocument[readPage.document] = readPage.text
	}

	return documentTextPerDocument
}
