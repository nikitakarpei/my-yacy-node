package judgedqueries_test

import (
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagecontents"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	missingPagesCapturingSwitch = "YACYDHTSEARCH_CAPTURE_MISSING_JUDGED_QUERY_PAGES"

	gradeOfAGonePage = 0
)

func TestCaptureTheMissingPagesOfTheJudgedQueries(t *testing.T) {
	if os.Getenv(missingPagesCapturingSwitch) == "" {
		t.Skipf("set %s to capture the missing pages of the judged queries over the web",
			missingPagesCapturingSwitch)
	}

	fetching := pageFetchingWithin(capturingPagesBudget)
	extraction := pageExtractionOfEveryFormat(t)
	for _, findingsFile := range recordedFindingsFiles(t) {
		storeCapturedPagesOf(t, findingsFile, fetching.missingPageCapturesOf(t, findingsFile))
		extraction.deriveMissingPageContentsOf(t, findingsFile)
		applyTheGradesOfCaptureFailuresOf(t, findingsFile)
	}
}

func (fetching pageFetching) missingPageCapturesOf(
	t *testing.T,
	findingsFile string,
) []pageCapture {
	t.Helper()

	recorded := recordedFindingsAt(t, findingsFile)
	missingDocuments := documentsWithoutAStoredPageAmong(
		judgedDocumentsAmong(t, recorded), storedPagePerAddressOf(t, recorded.Query),
	)

	return fetching.pageCapturesOf(t.Context(), missingDocuments)
}

func documentsWithoutAStoredPageAmong(
	documents []queryfindings.FoundDocument, storedPagePerAddress map[string]storedPage,
) []queryfindings.FoundDocument {
	documentsWithoutAStoredPage := make([]queryfindings.FoundDocument, 0, len(documents))
	for _, document := range documents {
		if _, stored := storedPagePerAddress[document.Address]; stored {
			continue
		}
		documentsWithoutAStoredPage = append(documentsWithoutAStoredPage, document)
	}

	return documentsWithoutAStoredPage
}

func storeCapturedPagesOf(t *testing.T, findingsFile string, pageCaptures []pageCapture) {
	t.Helper()

	if len(pageCaptures) == 0 {
		return
	}
	recorded := recordedFindingsAt(t, findingsFile)
	capturedPages := pagesAmong(pageCaptures)
	if len(capturedPages) > 0 {
		appendStoredPagesOf(t, recorded.Query, capturedPages)
	}
	writeRecordedFindingsFile(t, findingsFile,
		recorded.withCaptureFailures(captureFailurePerDocumentOf(pageCaptures)))
	t.Logf("%q captured the page of %d of its %d judged documents without a stored page",
		recorded.Query, len(capturedPages), len(pageCaptures))
}

func captureFailurePerDocumentOf(pageCaptures []pageCapture) map[yacymodel.URLHash]captureFailure {
	captureFailurePerDocument := make(map[yacymodel.URLHash]captureFailure, len(pageCaptures))
	for _, capture := range pageCaptures {
		captureFailurePerDocument[capture.document] = capture.failure
	}

	return captureFailurePerDocument
}

func (extraction pageExtraction) deriveMissingPageContentsOf(t *testing.T, findingsFile string) {
	t.Helper()

	recorded := recordedFindingsAt(t, findingsFile)
	findings := recorded.findings()
	pageContentsPerDocument := recorded.pageContentsPerDocument()
	derivedPageContents := extraction.pageContentsPerDocumentOf(
		t.Context(),
		queryreading.QueryFrom(recorded.Query, recorded.Language).Words,
		findings,
		storedPagePerAddressOf(t, recorded.Query),
	)
	amountOfDerivedDocuments := addMissingPageContents(pageContentsPerDocument, derivedPageContents)
	if amountOfDerivedDocuments == 0 {
		return
	}
	writeRecordedFindingsFile(t, findingsFile, recorded.withPageContentsReadAgain(
		findingsAndPageContentsOf(findings, pageContentsPerDocument),
	))
	t.Logf("%q derived the text of %d more documents", recorded.Query, amountOfDerivedDocuments)
}

func addMissingPageContents(
	pageContentsPerDocument map[yacymodel.URLHash]pagecontents.PageContents,
	derivedPageContents map[yacymodel.URLHash]pagecontents.PageContents,
) int {
	amountOfAddedDocuments := 0
	for document, pageContents := range derivedPageContents {
		if _, read := pageContentsPerDocument[document]; read {
			continue
		}
		pageContentsPerDocument[document] = pageContents
		amountOfAddedDocuments++
	}

	return amountOfAddedDocuments
}

func applyTheGradesOfCaptureFailuresOf(t *testing.T, findingsFile string) {
	t.Helper()

	recorded := recordedFindingsAt(t, findingsFile)
	gradePerFailure := map[captureFailure]yacymodel.Optional[int]{
		passingCaptureFailure: yacymodel.None[int](),
		goneCaptureFailure:    yacymodel.Some(gradeOfAGonePage),
	}
	gradePerDocument := map[yacymodel.URLHash]yacymodel.Optional[int]{}
	for document, failure := range recorded.captureFailurePerDocument() {
		if grade, ruled := gradePerFailure[failure]; ruled {
			gradePerDocument[document] = grade
		}
	}
	if amountOfChangedGrades := regradedDocumentsOf(
		t,
		recorded.Query,
		gradePerDocument,
	); amountOfChangedGrades > 0 {
		t.Logf(
			"%q changes %d grades by the reason its capture failed",
			recorded.Query,
			amountOfChangedGrades,
		)
	}
}

func regradedDocumentsOf(
	t *testing.T, query string, gradePerDocument map[yacymodel.URLHash]yacymodel.Optional[int],
) int {
	t.Helper()

	judgments := queryJudgmentsAt(t, queryJudgmentsFileOf(query))
	amountOfChangedGrades := 0
	for place, judged := range judgments.JudgedDocuments {
		grade, ruled := gradePerDocument[judged.Hash]
		if !ruled || grade == judged.Grade || (judged.Spam && !grade.Present()) {
			continue
		}
		judgments.JudgedDocuments[place].Grade = grade
		amountOfChangedGrades++
	}
	if amountOfChangedGrades > 0 {
		writeFixtureFile(t, queryJudgmentsFileOf(query), judgments)
	}

	return amountOfChangedGrades
}
