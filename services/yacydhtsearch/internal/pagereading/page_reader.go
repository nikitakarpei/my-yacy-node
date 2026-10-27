package pagereading

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
	pageclassificationhttpheader "github.com/nikitakarpei/yacy-rwi-node/pageclassification/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/htmlmeta"
	robotsmetahttpheader "github.com/nikitakarpei/yacy-rwi-node/robotsmeta/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
)

type pageReader struct {
	pageFetch            PageFetcher
	formatDerivations    FormatDerivations
	snippetLengthCeiling int
}

func (reader pageReader) read(
	ctx context.Context,
	queryWords []string,
	pageToRead PageToRead,
) pageReadResult {
	pageURL, err := canonicalurl.CanonicalURLOf(pageToRead.Address)
	if err != nil {
		return pageReadResult{document: pageToRead.Document, outcome: pageWasUnreachable}
	}
	fetchStartedAt := time.Now()
	landed, err := reader.pageFetch.Fetch(ctx, pageURL, pagefetch.PageVersion{})
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

	if ctx.Err() != nil {
		return pageReadResult{
			document:          pageToRead.Document,
			outcome:           pageWasOutOfBudget,
			timeSpentFetching: timeSpentFetching,
		}
	}
	readingStartedAt := time.Now()
	pageContents, outcome := reader.pageContentsOfTheFetchedPage(
		ctx, queryWords, landed.Outcome.Page, landed.URL,
	)
	if landed.URL != pageURL {
		pageContents.Address = landed.URL.String()
	}

	return pageReadResult{
		document:     pageToRead.Document,
		outcome:      outcome,
		pageContents: pageContents,
		spamVerdict: pageclassificationhttpheader.VerdictsFrom(
			landed.Outcome.Page.PageClassificationValue,
		)[pageclassification.Spam],
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

func (reader pageReader) pageContentsOfTheFetchedPage(
	ctx context.Context,
	queryWords []string,
	fetchedPage pagefetch.FetchedPage,
	pageURL canonicalurl.CanonicalURL,
) (pagecontents.PageContents, readOutcome) {
	if robotsRefusalsOf(fetchedPage).RefusesIndexing {
		return pagecontents.PageContents{}, pageRefusesIndexing
	}
	extractedDocument, err := documentextraction.DocumentFrom(
		ctx, fetchedPage.Body, fetchedPage.ContentType, pageURL,
	)
	if err != nil {
		return pagecontents.PageContents{}, readOutcomeFromAnExtractionFailure(err)
	}
	extractedPage, derived := reader.extractedPageFrom(ctx, extractedDocument, pageURL)
	if !derived {
		return pagecontents.PageContents{}, pageWasUnreadable
	}

	return pagecontents.PageContentsFrom(
		extractedPage,
		queryWords,
		reader.snippetLengthCeiling,
	), pageWasRead
}

func robotsRefusalsOf(fetchedPage pagefetch.FetchedPage) robotsmeta.Refusals {
	return robotsmetahttpheader.RefusalsOf(fetchedPage.RobotsTagValues).
		With(htmlmeta.RefusalsOf(fetchedPage.ContentType, fetchedPage.Body))
}

func readOutcomeFromAnExtractionFailure(extractionFailure error) readOutcome {
	if errors.Is(extractionFailure, documentextraction.ErrUnsupportedMediaType) {
		return pageWasOfAnUnsupportedKind
	}

	return pageWasUnreadable
}

func (reader pageReader) extractedPageFrom(
	ctx context.Context,
	extractedDocument documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) (pagecontents.ExtractedPage, bool) {
	readableText, readableTextDerived := reader.formatDerivations.BytesIn(
		ctx, documentextraction.FormatReadableText, extractedDocument, pageURL,
	)
	if readableTextDerived && len(bytes.TrimSpace(readableText)) > 0 {
		auxiliaryText, _ := reader.formatDerivations.BytesIn(
			ctx, documentextraction.FormatAuxiliaryText, extractedDocument, pageURL,
		)
		return pagecontents.ExtractedPage{
			Title:         extractedDocument.Title,
			Text:          string(readableText),
			AuxiliaryText: string(auxiliaryText),
			LinkCounts:    linkCountsOfTheExtractedDocument(extractedDocument),
		}, true
	}
	fullText, fullTextDerived := reader.formatDerivations.BytesIn(
		ctx, documentextraction.FormatFullText, extractedDocument, pageURL,
	)

	return pagecontents.ExtractedPage{
		Title:      extractedDocument.Title,
		Text:       string(fullText),
		LinkCounts: linkCountsOfTheExtractedDocument(extractedDocument),
	}, fullTextDerived
}

func linkCountsOfTheExtractedDocument(
	extractedDocument documentextraction.Document,
) pagecontents.LinkCounts {
	return pagecontents.LinkCounts{
		LocalLinks:    extractedDocument.LocalLinks,
		ExternalLinks: extractedDocument.ExternalLinks,
	}
}
