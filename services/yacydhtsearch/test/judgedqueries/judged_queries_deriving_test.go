package judgedqueries_test

import (
	"os"
	"testing"
)

const derivingSwitch = "YACYDHTSEARCH_DERIVE_JUDGED_QUERIES"

func TestDeriveTheJudgedQueriesFromTheStoredPageText(t *testing.T) {
	if os.Getenv(derivingSwitch) == "" {
		t.Skipf("set %s to derive the judged queries from the stored page text", derivingSwitch)
	}

	for _, answersFile := range recordedAnswersFiles(t) {
		deriveOneJudgedQueryFromTheStoredPageText(t, answersFile)
	}
}

func deriveOneJudgedQueryFromTheStoredPageText(t *testing.T, answersFile string) {
	t.Helper()

	answers := recordedAnswersInTheFile(t, answersFile)
	pageTextPerDocument := storedPageTextPerDocument(t, answers.Query)
	derived := answersCarryingThePageTextOfEachDocument(
		answers.Query, answers.answeredQuery(), pageTextPerDocument,
	)
	writeRecordedAnswersFile(t, answersFile, recordedAnswersOf(answers.Query, derived))
	judgments := queryJudgmentsOfTheDocumentsToJudge(
		answers.Query,
		derived,
		pageTextPerDocument,
		queryJudgmentsInTheFile(t, queryJudgmentsFileOf(answers.Query)),
	)
	writeFixtureFile(t, queryJudgmentsFileOf(answers.Query), judgments)
	t.Logf("%q derived the text of %d documents, %d documents wait for a grade",
		answers.Query, len(pageTextPerDocument), judgments.amountOfUngradedDocuments())
}
