package judgedqueries_test

import (
	"os"
	"testing"
)

const derivingSwitch = "YACYDHTSEARCH_DERIVE_JUDGED_QUERIES"

func TestDeriveTheJudgedQueriesFromTheStoredPages(t *testing.T) {
	if os.Getenv(derivingSwitch) == "" {
		t.Skipf("set %s to derive the judged queries from the stored pages", derivingSwitch)
	}

	extraction := pageExtractionOfTheFormats(t)
	for _, answersFile := range recordedAnswersFiles(t) {
		deriveOneJudgedQueryFromTheStoredPages(t, extraction, answersFile)
	}
}

func deriveOneJudgedQueryFromTheStoredPages(
	t *testing.T, extraction pageExtraction, answersFile string,
) {
	t.Helper()

	answers := recordedAnswersInTheFile(t, answersFile)
	answersAndTheirPages := extraction.answersWithTheContentsOfTheStoredPages(
		t.Context(),
		t,
		answers.Query,
		answers.answeredQuery(),
		storedPagePerAddress(t, answers.Query),
	)
	writeRecordedAnswersFile(
		t,
		answersFile,
		recordedAnswersOf(answers.Query, answersAndTheirPages),
	)
	judgments := queryJudgmentsRecordedFor(t, answers.Query).withTheDocumentsToJudgeIn(
		answersAndTheirPages.answeredQuery.WithTheContentsOfTheReadPages(
			answersAndTheirPages.pageContentsPerDocument,
		),
		answersAndTheirPages.pageContentsPerDocument,
	)
	writeFixtureFile(t, queryJudgmentsFileOf(answers.Query), judgments)
	t.Logf("%q derived the text of %d documents, %d documents wait for a grade",
		answers.Query,
		len(answersAndTheirPages.pageContentsPerDocument),
		judgments.amountOfUngradedDocuments())
}
