package judgedqueries_test

import (
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
)

const derivingSwitch = "YACYDHTSEARCH_DERIVE_JUDGED_QUERIES"

func TestDeriveTheJudgedQueriesFromTheStoredPages(t *testing.T) {
	if os.Getenv(derivingSwitch) == "" {
		t.Skipf("set %s to derive the judged queries from the stored pages", derivingSwitch)
	}

	extraction := pageExtractionOfEveryFormat(t)
	for _, findingsFile := range recordedFindingsFiles(t) {
		extraction.deriveJudgedQueryFrom(t, findingsFile)
	}
}

func (extraction pageExtraction) deriveJudgedQueryFrom(t *testing.T, findingsFile string) {
	t.Helper()

	recorded := recordedFindingsAt(t, findingsFile)
	findings := recorded.findings()
	findingsAndPageContents := findingsAndPageContentsOf(
		findings,
		extraction.pageContentsPerDocumentOf(
			t.Context(),
			queryreading.QueryFrom(recorded.Query, "").Words,
			findings,
			storedPagePerAddressOf(t, recorded.Query),
		),
	)
	writeRecordedFindingsFile(
		t, findingsFile, recorded.withPageContentsReadAgain(findingsAndPageContents),
	)
	judgments := writeJudgmentsOf(t, recorded.Query, findingsAndPageContents)
	t.Logf("%q derived the text of %d documents, %d documents wait for a grade",
		recorded.Query,
		len(findingsAndPageContents.pageContentsPerDocument),
		judgments.amountOfUngradedDocuments())
}
