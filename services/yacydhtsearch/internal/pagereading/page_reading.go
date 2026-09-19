// Package pagereading reads the page of each document one query puts first,
// all at once inside one read budget, and gives back the contents of the page of
// each document it could read: its text, and how many links of its own site and
// of other sites it holds. It takes the readable text of the page, and the whole text
// when the page holds no readable article. A page it cannot fetch, read, or
// finish inside the budget gives back nothing for its document. A page whose
// site answers that it is not found or gone gives back its document as gone.
// It follows the redirects of a page, and a page that moved gives back the
// address it moved to.
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
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documenttext"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageToRead struct {
	Document yacymodel.URLHash
	Address  string
}

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		knownVersion pagefetch.PageVersion,
	) (redirectfollowingfetch.LandedFetch, error)
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
	pageFetch            PageFetcher
	formatDerivations    FormatDerivations
	pageReadBudget       time.Duration
	snippetLengthCeiling int
	observer             PageReadingObserver
}

func New(
	pageFetch PageFetcher,
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

func (r Reading) ReadEachPage(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	pagesToRead []PageToRead,
) ReadPages {
	startedAt := time.Now()
	budgetedCtx, stopPageReadBudget := context.WithTimeout(ctx, r.pageReadBudget)
	defer stopPageReadBudget()

	pageReadResults := r.readEachPageAtOnce(budgetedCtx, queryWords, pagesToRead)
	r.observer.PageReadingPerformed(
		ctx,
		performedPageReadingFrom(pageReadResults, time.Since(startedAt)),
	)

	return readPagesFrom(pageReadResults)
}

func (r Reading) readEachPageAtOnce(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	pagesToRead []PageToRead,
) []pageReadResult {
	pageReadResults := make([]pageReadResult, len(pagesToRead))
	var pagesBeingRead sync.WaitGroup
	for place, pageToRead := range pagesToRead {
		pagesBeingRead.Add(1)
		go func() {
			defer pagesBeingRead.Done()
			pageReadResults[place] = r.readThePage(ctx, queryWords, pageToRead)
		}()
	}
	pagesBeingRead.Wait()

	return pageReadResults
}

func (r Reading) readThePage(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	pageToRead PageToRead,
) pageReadResult {
	pageURL, err := canonicalurl.CanonicalURLOf(pageToRead.Address)
	if err != nil {
		return pageReadResult{document: pageToRead.Document, outcome: pageWasUnreachable}
	}
	fetchStartedAt := time.Now()
	landed, err := r.pageFetch.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	timeSpentFetching := time.Since(fetchStartedAt)
	if err != nil {
		return pageReadResult{
			document:          pageToRead.Document,
			outcome:           readOutcomeFromAFetchFailure(err, ctx.Err()),
			timeSpentFetching: timeSpentFetching,
		}
	}
	if landed.Outcome.Status != pagefetch.FetchSucceeded {
		return pageReadResult{
			document:          pageToRead.Document,
			outcome:           readOutcomeFromAFetchStatus(landed.Outcome.Status),
			timeSpentFetching: timeSpentFetching,
		}
	}

	readingStartedAt := time.Now()
	pageContents, outcome := r.pageContentsOfTheFetchedPage(
		ctx, queryWords, landed.Outcome.Page, landed.URL,
	)
	if landed.URL != pageURL {
		pageContents.Text.Address = landed.URL.String()
	}

	return pageReadResult{
		document:          pageToRead.Document,
		outcome:           outcome,
		pageContents:      pageContents,
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
	switch status {
	case pagefetch.FetchDeadlinePassed:
		return pageWasOutOfBudget
	case pagefetch.FetchGone:
		return pageWasGone
	default:
		return pageWasRefused
	}
}

func (r Reading) pageContentsOfTheFetchedPage(
	ctx context.Context,
	queryWords []yacymodel.Hash,
	fetchedPage pagefetch.FetchedPage,
	pageURL canonicalurl.CanonicalURL,
) (queryanswers.PageContents, readOutcome) {
	extractedDocument, err := documentextraction.DocumentFrom(
		ctx, fetchedPage.Body, fetchedPage.ContentType, pageURL,
	)
	if err != nil {
		return queryanswers.PageContents{}, readOutcomeFromAnExtractionFailure(err)
	}
	text, derived := r.textOfTheExtractedDocument(ctx, extractedDocument, pageURL)
	if !derived {
		return queryanswers.PageContents{}, pageWasUnreadable
	}

	return queryanswers.PageContents{
		Text: documenttext.DocumentTextFrom(
			extractedDocument.Title, string(text), queryWords, r.snippetLengthCeiling,
		),
		LinkCounts: linkCountsOfTheExtractedDocument(extractedDocument),
	}, pageWasRead
}

func linkCountsOfTheExtractedDocument(
	extractedDocument documentextraction.Document,
) queryanswers.LinkCounts {
	return queryanswers.LinkCounts{
		LocalLinks:    extractedDocument.LocalLinks,
		ExternalLinks: extractedDocument.ExternalLinks,
	}
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
