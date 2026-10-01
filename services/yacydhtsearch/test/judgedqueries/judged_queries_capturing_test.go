package judgedqueries_test

import (
	"os"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

const (
	capturingSwitch      = "YACYDHTSEARCH_CAPTURE_JUDGED_QUERY_PAGES"
	capturingPagesBudget = 90 * time.Second
)

func TestCaptureThePagesOfTheJudgedQueries(t *testing.T) {
	if os.Getenv(capturingSwitch) == "" {
		t.Skipf("set %s to capture the pages of the judged queries over the web", capturingSwitch)
	}

	fetching := pageFetchingWithin(capturingPagesBudget)
	for _, findingsFile := range recordedFindingsFiles(t) {
		fetching.capturePagesOf(t, findingsFile)
	}
}

func (fetching pageFetching) capturePagesOf(t *testing.T, findingsFile string) {
	t.Helper()

	recorded := recordedFindingsAt(t, findingsFile)
	judgedDocuments := judgedDocumentsAmong(t, recorded)
	pages := fetching.fetchedPagesOf(t.Context(), judgedDocuments)
	writeStoredPagesOf(t, recorded.Query, pages)
	t.Logf("%q captured the page of %d of the %d judged documents",
		recorded.Query, len(pages), len(judgedDocuments))
}

func judgedDocumentsAmong(
	t *testing.T, recorded recordedFindings,
) []queryfindings.FoundDocument {
	t.Helper()

	judgedDocumentPerHash := queryJudgmentsAt(t, queryJudgmentsFileOf(recorded.Query)).
		judgedDocumentPerHash()
	foundDocuments := recorded.findings().FoundDocuments
	judgedDocuments := make([]queryfindings.FoundDocument, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		if _, judged := judgedDocumentPerHash[foundDocument.Hash]; !judged {
			continue
		}
		judgedDocuments = append(judgedDocuments, foundDocument)
	}

	return judgedDocuments
}
