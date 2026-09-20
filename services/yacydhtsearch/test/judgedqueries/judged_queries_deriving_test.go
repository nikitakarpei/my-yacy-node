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
	saturated := extraction.answersSaturatedWithTheStoredPages(
		t.Context(),
		t,
		answers.Query,
		answers.answeredQuery(),
		storedPagePerAddress(t, answers.Query),
	)
	writeRecordedAnswersFile(t, answersFile, recordedAnswersOf(answers.Query, saturated.answers))
	judgments := queryJudgmentsOfTheDocumentsToJudge(
		answers.Query,
		saturated.answers,
		saturated.pageContentsPerDocument,
		queryJudgmentsInTheFile(t, queryJudgmentsFileOf(answers.Query)),
	)
	writeFixtureFile(t, queryJudgmentsFileOf(answers.Query), judgments)
	t.Logf("%q derived the text of %d documents, %d documents wait for a grade",
		answers.Query,
		len(saturated.pageContentsPerDocument),
		judgments.amountOfUngradedDocuments())
}
