package judgedqueries_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/hostdiscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/peerorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
)

const (
	leastLiftOfTheOrderingOfTheServiceOverThePeerOrdering = 0.27
	toleranceBelowTheAcceptedMeanGain                     = 0.02
)

type itemsOrdering interface {
	OrderedItemsOf(answers peeranswers.AnsweredQuery) []peeranswers.AnsweredItem
}

func TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries(t *testing.T) {
	t.Parallel()

	judged := judgedQueriesRecorded(t)
	gainOfTheOrderingOfTheService := gainPerJudgedQueryOf(
		orderingOfTheServiceFrom(relevance.DefaultScoreWeights()), judged,
	)
	acceptedGain := acceptedGainPerJudgedQueryInTheFile(t, acceptedGainFile)

	reportTheGainOfEachJudgedQuery(t, judged, gainOfTheOrderingOfTheService)
	failIfTheLiftOverThePeerOrderingFallsShort(t, judged)
	failIfTheMeanGainFallsBelowTheAcceptedGain(t, gainOfTheOrderingOfTheService, acceptedGain)
	failIfAJudgedQueryFellToNoGain(t, gainOfTheOrderingOfTheService, acceptedGain)
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

func orderingOfTheServiceFrom(scoreWeights relevance.ScoreWeights) hostdiscount.Ordering {
	return hostdiscount.New(relevance.New(scoreWeights))
}

func reportTheGainOfEachJudgedQuery(
	t *testing.T, judged []judgedQuery, gainOfTheOrderingOfTheService gainPerJudgedQuery,
) {
	t.Helper()

	relevanceOrdering := relevance.New(relevance.DefaultScoreWeights())
	for _, judgedQuery := range judged {
		orderedItems := hostdiscount.New(relevanceOrdering).OrderedItemsOf(judgedQuery.answers)
		t.Logf(
			"%q: host discount %.4f, relevance %.4f, peer order %.4f, %d ungraded documents "+
				"dropped",
			judgedQuery.query,
			gainOfTheOrderingOfTheService[judgedQuery.query],
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
				relevanceOrdering.OrderedItemsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
				peerorder.Ordering{}.OrderedItemsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.amountOfUngradedItemsAmong(orderedItems),
		)
	}
	t.Logf(
		"the mean over %d judged queries: host discount %.4f, relevance %.4f, peer order %.4f",
		len(judged),
		meanNormalizedGainDiscountedPerHostOf(hostdiscount.New(relevanceOrdering), judged),
		meanNormalizedGainDiscountedPerHostOf(relevanceOrdering, judged),
		meanNormalizedGainDiscountedPerHostOf(peerorder.Ordering{}, judged),
	)
	t.Logf(
		"the ordering of the service reaches no gain on %d of %d judged queries",
		gainOfTheOrderingOfTheService.amountOfQueriesWithoutGain(),
		len(judged),
	)
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

func failIfTheLiftOverThePeerOrderingFallsShort(t *testing.T, judged []judgedQuery) {
	t.Helper()

	meanGainOfTheOrderingOfTheService := meanNormalizedGainDiscountedPerHostOf(
		orderingOfTheServiceFrom(relevance.DefaultScoreWeights()), judged,
	)
	meanGainOfThePeerOrdering := meanNormalizedGainDiscountedPerHostOf(
		peerorder.Ordering{}, judged,
	)
	if meanGainOfTheOrderingOfTheService-meanGainOfThePeerOrdering >=
		leastLiftOfTheOrderingOfTheServiceOverThePeerOrdering {
		return
	}
	t.Errorf(
		"the ordering of the service lifts the mean gain from %.4f to %.4f over %d judged "+
			"queries, want a lift of at least %.2f",
		meanGainOfThePeerOrdering,
		meanGainOfTheOrderingOfTheService,
		len(judged),
		leastLiftOfTheOrderingOfTheServiceOverThePeerOrdering,
	)
}

func failIfTheMeanGainFallsBelowTheAcceptedGain(
	t *testing.T, gainOfTheOrderingOfTheService, acceptedGain gainPerJudgedQuery,
) {
	t.Helper()

	meanGainOfTheOrderingOfTheService := gainOfTheOrderingOfTheService.
		meanGainOverTheQueriesIn(acceptedGain)
	meanOfTheAcceptedGain := acceptedGain.
		meanGainOverTheQueriesIn(gainOfTheOrderingOfTheService)
	if meanGainOfTheOrderingOfTheService >=
		meanOfTheAcceptedGain-toleranceBelowTheAcceptedMeanGain {
		return
	}
	t.Errorf(
		"the ordering of the service reaches a mean gain of %.4f over the judged queries of "+
			"the baseline, want at least the accepted mean %.4f less the tolerance %.2f",
		meanGainOfTheOrderingOfTheService,
		meanOfTheAcceptedGain,
		toleranceBelowTheAcceptedMeanGain,
	)
}

func failIfAJudgedQueryFellToNoGain(
	t *testing.T, gainOfTheOrderingOfTheService, acceptedGain gainPerJudgedQuery,
) {
	t.Helper()

	for _, query := range slices.Sorted(maps.Keys(gainOfTheOrderingOfTheService)) {
		accepted, inTheBaseline := acceptedGain[query]
		if !inTheBaseline || accepted == 0 || gainOfTheOrderingOfTheService[query] > 0 {
			continue
		}
		t.Errorf(
			"the ordering of the service reaches no gain on %q, the baseline accepts %.4f",
			query,
			accepted,
		)
	}
}
