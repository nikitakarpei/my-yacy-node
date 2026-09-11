package judgedqueries_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/peerorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
)

const (
	meanGainFloorOfTheRelevanceOrdering                = 0.80
	leastLiftOfTheRelevanceOrderingOverThePeerOrdering = 0.20
)

type itemsOrdering interface {
	OrderedItemsOf(answers peeranswers.AnsweredQuery) []peeranswers.AnsweredItem
}

func TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries(t *testing.T) {
	t.Parallel()

	judged := judgedQueriesRecorded(t)

	meanGainOfTheRelevanceOrdering := meanNormalizedGainOf(relevance.Ordering{}, judged)
	meanGainOfThePeerOrdering := meanNormalizedGainOf(peerorder.Ordering{}, judged)

	if meanGainOfTheRelevanceOrdering < meanGainFloorOfTheRelevanceOrdering {
		t.Errorf(
			"the relevance ordering reaches a mean gain of %.4f over %d judged queries, "+
				"want at least %.2f; the peer ordering reaches %.4f",
			meanGainOfTheRelevanceOrdering,
			len(judged),
			meanGainFloorOfTheRelevanceOrdering,
			meanGainOfThePeerOrdering,
		)
	}
	if meanGainOfTheRelevanceOrdering-meanGainOfThePeerOrdering <
		leastLiftOfTheRelevanceOrderingOverThePeerOrdering {
		t.Errorf(
			"the relevance ordering lifts the mean gain from %.4f to %.4f over %d judged "+
				"queries, want a lift of at least %.2f",
			meanGainOfThePeerOrdering,
			meanGainOfTheRelevanceOrdering,
			len(judged),
			leastLiftOfTheRelevanceOrderingOverThePeerOrdering,
		)
	}
}

type judgedQuery struct {
	answers         peeranswers.AnsweredQuery
	gradedDocuments gradedDocuments
}

func judgedQueriesRecorded(t *testing.T) []judgedQuery {
	t.Helper()

	var judged []judgedQuery
	for _, answersFile := range recordedAnswersFiles(t) {
		graded := gradedDocumentsOfTheAnswersFile(t, answersFile)
		if !graded.holdARelevantDocument() {
			continue
		}
		judged = append(judged, judgedQuery{
			answers:         recordedAnswersInTheFile(t, answersFile).answeredQuery(),
			gradedDocuments: graded,
		})
	}

	return judged
}

func gradedDocumentsOfTheAnswersFile(t *testing.T, answersFile string) gradedDocuments {
	t.Helper()

	judgmentsFile := filepath.Join(queryJudgmentsDirectory, filepath.Base(answersFile))
	if _, err := os.Stat(judgmentsFile); err != nil {
		t.Fatalf("read %s: %v", judgmentsFile, err)
	}

	return queryJudgmentsInTheFile(t, judgmentsFile).gradeOfEachDocument()
}

func meanNormalizedGainOf(ordering itemsOrdering, judged []judgedQuery) float64 {
	sumOfNormalizedGains := 0.0
	for _, judgedQuery := range judged {
		sumOfNormalizedGains += judgedQuery.gradedDocuments.normalizedGainOf(
			ordering.OrderedItemsOf(judgedQuery.answers),
		)
	}

	return sumOfNormalizedGains / float64(len(judged))
}
