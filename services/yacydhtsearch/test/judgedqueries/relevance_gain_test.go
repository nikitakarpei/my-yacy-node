package judgedqueries_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/hostturns"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/peerorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
)

const (
	meanGainFloorOfTheOrderingOfTheService                = 0.74
	leastLiftOfTheOrderingOfTheServiceOverThePeerOrdering = 0.27
)

type itemsOrdering interface {
	OrderedItemsOf(answers peeranswers.AnsweredQuery) []peeranswers.AnsweredItem
}

func TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries(t *testing.T) {
	t.Parallel()

	judged := judgedQueriesRecorded(t)

	meanGainOfTheOrderingOfTheService := meanNormalizedGainDiscountedPerHostOf(
		hostturns.New(relevance.Ordering{}), judged,
	)
	meanGainOfThePeerOrdering := meanNormalizedGainDiscountedPerHostOf(
		peerorder.Ordering{}, judged,
	)
	reportTheGainOfEachJudgedQuery(t, judged)

	if meanGainOfTheOrderingOfTheService < meanGainFloorOfTheOrderingOfTheService {
		t.Errorf(
			"the ordering of the service reaches a mean gain of %.4f over %d judged queries, "+
				"want at least %.2f; the peer ordering reaches %.4f",
			meanGainOfTheOrderingOfTheService,
			len(judged),
			meanGainFloorOfTheOrderingOfTheService,
			meanGainOfThePeerOrdering,
		)
	}
	if meanGainOfTheOrderingOfTheService-meanGainOfThePeerOrdering <
		leastLiftOfTheOrderingOfTheServiceOverThePeerOrdering {
		t.Errorf(
			"the ordering of the service lifts the mean gain from %.4f to %.4f over %d judged "+
				"queries, want a lift of at least %.2f",
			meanGainOfThePeerOrdering,
			meanGainOfTheOrderingOfTheService,
			len(judged),
			leastLiftOfTheOrderingOfTheServiceOverThePeerOrdering,
		)
	}
}

type judgedQuery struct {
	query           string
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
		answers := recordedAnswersInTheFile(t, answersFile)
		judged = append(judged, judgedQuery{
			query:           answers.Query,
			answers:         answers.answeredQuery(),
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

	return queryJudgmentsInTheFile(t, judgmentsFile).gradedDocumentsOfTheQuery()
}

func meanNormalizedGainDiscountedPerHostOf(
	ordering itemsOrdering, judged []judgedQuery,
) float64 {
	sumOfNormalizedGains := 0.0
	for _, judgedQuery := range judged {
		sumOfNormalizedGains += judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
			ordering.OrderedItemsOf(judgedQuery.answers),
		)
	}

	return sumOfNormalizedGains / float64(len(judged))
}

func reportTheGainOfEachJudgedQuery(t *testing.T, judged []judgedQuery) {
	t.Helper()

	for _, judgedQuery := range judged {
		orderedItems := hostturns.New(relevance.Ordering{}).OrderedItemsOf(judgedQuery.answers)
		t.Logf(
			"%q: host turns %.4f, relevance %.4f, peer order %.4f, %d ungraded documents dropped",
			judgedQuery.query,
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(orderedItems),
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
				relevance.Ordering{}.OrderedItemsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
				peerorder.Ordering{}.OrderedItemsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.amountOfUngradedItemsAmong(orderedItems),
		)
	}
	t.Logf(
		"the mean over %d judged queries: host turns %.4f, relevance %.4f, peer order %.4f",
		len(judged),
		meanNormalizedGainDiscountedPerHostOf(hostturns.New(relevance.Ordering{}), judged),
		meanNormalizedGainDiscountedPerHostOf(relevance.Ordering{}, judged),
		meanNormalizedGainDiscountedPerHostOf(peerorder.Ordering{}, judged),
	)
}
