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

	extraction := pageExtractionOfEveryFormat(t)
	for _, answersFile := range recordedAnswersFiles(t) {
		extraction.deriveJudgedQueryFrom(t, answersFile)
	}
}

func (extraction pageExtraction) deriveJudgedQueryFrom(t *testing.T, answersFile string) {
	t.Helper()

	recorded := recordedAnswersAt(t, answersFile)
	answers := recorded.answers()
	answersAndPageContents := answersAndPageContentsOf(
		answers,
		extraction.pageContentsPerDocumentOf(
			t.Context(), answers, storedPagePerAddressOf(t, recorded.Query),
		),
	)
	writeRecordedAnswersFile(
		t, answersFile, recorded.withPageContentsReadAgain(answersAndPageContents),
	)
	judgments := writeJudgmentsOf(t, recorded.Query, answersAndPageContents)
	t.Logf("%q derived the text of %d documents, %d documents wait for a grade",
		recorded.Query,
		len(answersAndPageContents.pageContentsPerDocument),
		judgments.amountOfUngradedDocuments())
}
