package pagereading

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/htmlmeta"
	robotsmetahttpheader "github.com/nikitakarpei/yacy-rwi-node/robotsmeta/httpheader"
	spamassessmenthttpheader "github.com/nikitakarpei/yacy-rwi-node/spamassessment/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type pageReader struct {
	pageFetch            PageFetcher
	formatDerivations    FormatDerivations
	snippetLengthCeiling int
}

func (reader pageReader) read(
	ctx context.Context,
	queryWords []yacymodel.Hash,
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
		spamVerdict: spamassessmenthttpheader.VerdictFrom(
			landed.Outcome.Page.SpamAssessmentValue,
		),
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
	queryWords []yacymodel.Hash,
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
	text, derived := reader.textOfTheExtractedDocument(ctx, extractedDocument, pageURL)
	if !derived {
		return pagecontents.PageContents{}, pageWasUnreadable
	}

	return pagecontents.PageContentsFrom(
		extractedDocument.Title,
		string(text),
		linkCountsOfTheExtractedDocument(extractedDocument),
		queryWords,
		reader.snippetLengthCeiling,
	), pageWasRead
}

func robotsRefusalsOf(fetchedPage pagefetch.FetchedPage) robotsmeta.Refusals {
	return robotsmetahttpheader.RefusalsOf(fetchedPage.RobotsTagValues).
		With(htmlmeta.RefusalsOf(fetchedPage.ContentType, fetchedPage.Body))
}

func linkCountsOfTheExtractedDocument(
	extractedDocument documentextraction.Document,
) pagecontents.LinkCounts {
	return pagecontents.LinkCounts{
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

func (reader pageReader) textOfTheExtractedDocument(
	ctx context.Context,
	extractedDocument documentextraction.Document,
	pageURL canonicalurl.CanonicalURL,
) ([]byte, bool) {
	readableText, readableTextDerived := reader.formatDerivations.BodyIn(
		ctx, documentextraction.FormatReadableText, extractedDocument, pageURL,
	)
	if readableTextDerived && len(bytes.TrimSpace(readableText)) > 0 {
		return readableText, true
	}

	return reader.formatDerivations.BodyIn(
		ctx, documentextraction.FormatFullText, extractedDocument, pageURL,
	)
}
