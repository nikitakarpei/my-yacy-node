package judgedqueries_test

import (
	"os"
	"testing"
)

const poolingSwitch = "YACYDHTSEARCH_POOL_JUDGED_QUERIES"

func TestPoolTheJudgedQueriesAgain(t *testing.T) {
	if os.Getenv(poolingSwitch) == "" {
		t.Skipf("set %s to pool the judged queries again", poolingSwitch)
	}

	for _, answersFile := range recordedAnswersFiles(t) {
		poolOneJudgedQueryAgain(t, answersFile)
	}
}

func poolOneJudgedQueryAgain(t *testing.T, answersFile string) {
	t.Helper()

	answers := recordedAnswersInTheFile(t, answersFile)
	pooledDocuments := pooledDocumentsOf(answers.answeredQuery())
	judgmentsFile := queryJudgmentsFileOf(answers.Query)
	judgments := queryJudgmentsOfThePool(
		answers.Query, pooledDocuments, queryJudgmentsInTheFile(t, judgmentsFile),
	)
	writeFixtureFile(t, judgmentsFile, judgments)
	t.Logf("%q pooled %d documents, %d of them ungraded",
		answers.Query, len(pooledDocuments), judgments.amountOfUngradedDocuments())
}
