package judgedqueries_test

import (
	"os"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
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
	for _, answersFile := range recordedAnswersFiles(t) {
		fetching.capturePagesOf(t, answersFile)
	}
}

func (fetching pageFetching) capturePagesOf(t *testing.T, answersFile string) {
	t.Helper()

	recorded := recordedAnswersAt(t, answersFile)
	documentsToCapture := judgedDocumentsAmong(t, recorded)
	pages := fetching.fetchedPagesOf(t.Context(), documentsToCapture)
	writeStoredPagesOf(t, recorded.Query, pages)
	t.Logf("%q captured the page of %d of the %d judged documents",
		recorded.Query, len(pages), len(documentsToCapture))
}

func judgedDocumentsAmong(
	t *testing.T, recorded recordedAnswers,
) []queryanswers.FoundDocument {
	t.Helper()

	judgedDocumentPerHash := queryJudgmentsAt(t, queryJudgmentsFileOf(recorded.Query)).
		judgedDocumentPerHash()
	foundDocuments := recorded.answers().FoundDocuments
	judgedDocuments := make([]queryanswers.FoundDocument, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		if _, judged := judgedDocumentPerHash[foundDocument.Hash]; !judged {
			continue
		}
		judgedDocuments = append(judgedDocuments, foundDocument)
	}

	return judgedDocuments
}
