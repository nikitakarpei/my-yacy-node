// Package pagereading reads the pages of the documents one query puts first. One
// run per query starts each page once, at any time, and gives back, inside one
// budget, the text, link counts and spam verdict of the pages it is asked for. It
// withdraws the documents whose pages are gone or refuse indexing.
package pagereading

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta"
	"github.com/nikitakarpei/yacy-rwi-node/robotsmeta/htmlmeta"
	robotsmetahttpheader "github.com/nikitakarpei/yacy-rwi-node/robotsmeta/httpheader"
	spamassessmenthttpheader "github.com/nikitakarpei/yacy-rwi-node/spamassessment/httpheader"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
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

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type Reading struct {
	pageFetch            PageFetcher
	formatDerivations    FormatDerivations
	pageReadBudget       time.Duration
	cutoff               PageReadCutoff
	clock                Clock
	snippetLengthCeiling int
	observer             PageReadingObserver
}

//nolint:revive // argument-limit: the reading takes its fetch, formats, budget, cutoff, clock, snippet ceiling and observer
func New(
	pageFetch PageFetcher,
	formatDerivations FormatDerivations,
	pageReadBudget time.Duration,
	cutoff PageReadCutoff,
	clock Clock,
	snippetLengthCeiling int,
	observer PageReadingObserver,
) Reading {
	return Reading{
		pageFetch:            pageFetch,
		formatDerivations:    formatDerivations,
		pageReadBudget:       pageReadBudget,
		cutoff:               cutoff,
		clock:                clock,
		snippetLengthCeiling: snippetLengthCeiling,
		observer:             observer,
	}
}

func (r Reading) Start(queryWords []yacymodel.Hash) *Run {
	return newRun(func(ctx context.Context, pageToRead PageToRead) pageReadResult {
		return r.pageReadResultOf(ctx, queryWords, pageToRead)
	})
}

func (r Reading) pageReadResultOf(
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

func (r Reading) pageContentsOfTheFetchedPage(
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
	text, derived := r.textOfTheExtractedDocument(ctx, extractedDocument, pageURL)
	if !derived {
		return pagecontents.PageContents{}, pageWasUnreadable
	}

	return pagecontents.PageContentsFrom(
		extractedDocument.Title,
		string(text),
		linkCountsOfTheExtractedDocument(extractedDocument),
		queryWords,
		r.snippetLengthCeiling,
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

func (r Reading) ReadPagesAmong(
	ctx context.Context,
	run *Run,
	pagesWanted []PageToRead,
) ReadPages {
	startedAt := time.Now()
	run.Read(ctx, pagesWanted)
	pageReadResults := r.pageReadResultsWithinTheBudget(
		ctx, run.pageReadResultsAmong(pagesWanted), pagesWanted,
	)
	r.reportPageReading(
		ctx,
		performedPageReadingFrom(
			pageReadResults,
			run.amountOfPagesUnwanted(pagesWanted),
			time.Since(startedAt),
		),
	)

	return readPagesFrom(pageReadResults)
}

func (r Reading) pageReadResultsWithinTheBudget(
	ctx context.Context,
	settlingResults <-chan pageReadResult,
	pagesWanted []PageToRead,
) []pageReadResult {
	budgetedCtx, stopPageReadBudget := context.WithTimeout(ctx, r.pageReadBudget)
	defer stopPageReadBudget()

	return r.cutoff.pageReadResultsFrom(r.clock, budgetedCtx.Done(), settlingResults, pagesWanted)
}

func (r Reading) reportPageReading(ctx context.Context, pageReading PerformedPageReading) {
	r.observer.PageReadingPerformed(ctx, pageReading)
}
