package judgedqueries_test

import (
	"os"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const (
	capturingSwitch   = "YACYDHTSEARCH_CAPTURE_JUDGED_QUERY_PAGES"
	pageCaptureBudget = 90 * time.Second
)

func TestCaptureThePagesOfTheJudgedQueries(t *testing.T) {
	if os.Getenv(capturingSwitch) == "" {
		t.Skipf("set %s to capture the pages of the judged queries over the web", capturingSwitch)
	}

	fetching := pageFetchingOverTheWeb(pageCaptureBudget)
	for _, answersFile := range recordedAnswersFiles(t) {
		captureThePagesOfOneJudgedQuery(t, fetching, answersFile)
	}
}

func captureThePagesOfOneJudgedQuery(
	t *testing.T, fetching pageFetching, answersFile string,
) {
	t.Helper()

	answers := recordedAnswersInTheFile(t, answersFile)
	documentsToCapture := documentsJudgedOf(t, answers)
	pages := fetching.fetchedPagesOf(t.Context(), documentsToCapture)
	storePagesOfTheQuery(t, answers.Query, pages)
	t.Logf("%q captured the page of %d of the %d judged documents",
		answers.Query, len(pages), len(documentsToCapture))
}

func documentsJudgedOf(
	t *testing.T, answers recordedAnswers,
) []queryanswers.FoundDocument {
	t.Helper()

	judgedDocumentPerHash := queryJudgmentsInTheFile(t, queryJudgmentsFileOf(answers.Query)).
		judgedDocumentPerHash()
	foundDocuments := answers.answeredQuery().FoundDocuments
	judged := make([]queryanswers.FoundDocument, 0, len(foundDocuments))
	for _, foundDocument := range foundDocuments {
		if _, named := judgedDocumentPerHash[foundDocument.Hash]; !named {
			continue
		}
		judged = append(judged, foundDocument)
	}

	return judged
}
