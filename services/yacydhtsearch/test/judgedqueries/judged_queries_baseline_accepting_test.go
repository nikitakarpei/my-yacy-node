package judgedqueries_test

import (
	"os"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
)

const acceptingSwitch = "YACYDHTSEARCH_ACCEPT_JUDGED_QUERIES_BASELINE"

func TestAcceptTheGainOfEachJudgedQueryAsTheBaseline(t *testing.T) {
	if os.Getenv(acceptingSwitch) == "" {
		t.Skipf("set %s to accept the gain of each judged query as the baseline", acceptingSwitch)
	}

	judged := judgedQueriesRecorded(t)
	acceptedGain := gainPerJudgedQueryOf(
		orderingOfTheServiceFrom(documentrelevance.DefaultScoreWeights()), judged,
	)
	writeFixtureFile(t, acceptedGainFile, acceptedGain)
	t.Logf(
		"the baseline accepts the gain of %d judged queries, mean %.4f, %d without gain",
		len(acceptedGain),
		acceptedGain.meanGainOverTheQueriesIn(acceptedGain),
		acceptedGain.amountOfQueriesWithoutGain(),
	)
}
