package judgedqueries_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/hostdiscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const (
	leastLiftOfTheOrderingOfTheServiceOverThePeerOrdering = 0.27
	toleranceBelowTheAcceptedMeanGain                     = 0.02
)

type itemsOrdering interface {
	OrderedItemsOf(answers queryanswers.AnsweredQuery) []queryanswers.AnsweredItem
}

func TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries(t *testing.T) {
	t.Parallel()

	judged := judgedQueriesRecorded(t)
	gainOfTheOrderingOfTheService := gainPerJudgedQueryOf(
		orderingOfTheServiceFrom(documentrelevance.DefaultScoreWeights()), judged,
	)
	acceptedGain := acceptedGainPerJudgedQueryInTheFile(t, acceptedGainFile)

	reportTheGainOfEachJudgedQuery(t, judged, gainOfTheOrderingOfTheService)
	reportTheMeanGainUnderEachDiscount(t, judged)
	failIfTheLiftOverThePeerOrderingFallsShort(t, judged)
	failIfTheMeanGainFallsBelowTheAcceptedGain(t, gainOfTheOrderingOfTheService, acceptedGain)
	failIfAJudgedQueryFellToNoGain(t, gainOfTheOrderingOfTheService, acceptedGain)
}

type judgedQuery struct {
	query           string
	answers         queryanswers.AnsweredQuery
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

	judgmentsFile := filepath.Join(
		queryJudgmentsDirectory,
		strings.TrimSuffix(filepath.Base(answersFile), recordedAnswersFileSuffix)+
			queryJudgmentsFileSuffix,
	)
	if _, err := os.Stat(judgmentsFile); err != nil {
		t.Fatalf("read %s: %v", judgmentsFile, err)
	}

	return queryJudgmentsInTheFile(t, judgmentsFile).gradedDocumentsOfTheQuery()
}

func orderingOfTheServiceFrom(
	scoreWeights documentrelevance.ScoreWeights,
) hostdiscount.Ordering {
	return hostdiscount.New(documentrelevance.New(scoreWeights))
}

type orderingOfThePeerRankings struct{}

func (orderingOfThePeerRankings) OrderedItemsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.AnsweredItem {
	return answers.ItemOfEachAnsweredDocument()
}

func reportTheGainOfEachJudgedQuery(
	t *testing.T, judged []judgedQuery, gainOfTheOrderingOfTheService gainPerJudgedQuery,
) {
	t.Helper()

	documentRelevance := documentrelevance.New(documentrelevance.DefaultScoreWeights())
	relevanceOrdering := relevance.New(documentRelevance)
	for _, judgedQuery := range judged {
		orderedItems := hostdiscount.New(documentRelevance).OrderedItemsOf(judgedQuery.answers)
		t.Logf(
			"%q: the gain per subtopic of the host discount %.4f, of the relevance %.4f, "+
				"of the peer order %.4f, %d ungraded documents dropped",
			judgedQuery.query,
			gainOfTheOrderingOfTheService[judgedQuery.query],
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerSubtopicOf(
				relevanceOrdering.OrderedItemsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerSubtopicOf(
				orderingOfThePeerRankings{}.OrderedItemsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.amountOfUngradedItemsAmong(orderedItems),
		)
		t.Logf(
			"%q: the ordering of the service gains %.4f with no discount",
			judgedQuery.query,
			judgedQuery.gradedDocuments.normalizedGainWithNoDiscountOf(orderedItems),
		)
	}
	t.Logf(
		"the ordering of the service reaches no gain on %d of %d judged queries",
		gainOfTheOrderingOfTheService.amountOfQueriesWithoutGain(),
		len(judged),
	)
}

type orderingToReport struct {
	name     string
	ordering itemsOrdering
}

func reportTheMeanGainUnderEachDiscount(t *testing.T, judged []judgedQuery) {
	t.Helper()

	documentRelevance := documentrelevance.New(documentrelevance.DefaultScoreWeights())
	orderingsToReport := []orderingToReport{
		{name: "host discount", ordering: hostdiscount.New(documentRelevance)},
		{name: "relevance", ordering: relevance.New(documentRelevance)},
		{name: "peer order", ordering: orderingOfThePeerRankings{}},
	}
	for _, toReport := range orderingsToReport {
		t.Logf(
			"the mean over %d judged queries of the %s: %.4f per subtopic, "+
				"%.4f with no discount",
			len(judged),
			toReport.name,
			meanNormalizedGainOf(
				toReport.ordering, judged, gradedDocuments.normalizedGainDiscountedPerSubtopicOf,
			),
			meanNormalizedGainOf(
				toReport.ordering, judged, gradedDocuments.normalizedGainWithNoDiscountOf,
			),
		)
	}
}

func meanNormalizedGainDiscountedPerSubtopicOf(
	ordering itemsOrdering, judged []judgedQuery,
) float64 {
	return meanNormalizedGainOf(
		ordering, judged, gradedDocuments.normalizedGainDiscountedPerSubtopicOf,
	)
}

func meanNormalizedGainOf(
	ordering itemsOrdering,
	judged []judgedQuery,
	normalizedGainOf func(gradedDocuments, []queryanswers.AnsweredItem) float64,
) float64 {
	sumOfNormalizedGains := 0.0
	for _, judgedQuery := range judged {
		sumOfNormalizedGains += normalizedGainOf(
			judgedQuery.gradedDocuments, ordering.OrderedItemsOf(judgedQuery.answers),
		)
	}

	return sumOfNormalizedGains / float64(len(judged))
}

func failIfTheLiftOverThePeerOrderingFallsShort(t *testing.T, judged []judgedQuery) {
	t.Helper()

	meanGainOfTheOrderingOfTheService := meanNormalizedGainDiscountedPerSubtopicOf(
		orderingOfTheServiceFrom(documentrelevance.DefaultScoreWeights()), judged,
	)
	meanGainOfThePeerOrdering := meanNormalizedGainDiscountedPerSubtopicOf(
		orderingOfThePeerRankings{}, judged,
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
