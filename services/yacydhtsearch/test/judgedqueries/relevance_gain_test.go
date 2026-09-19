package judgedqueries_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/hostdiscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const (
	leastLiftOfTheOrderingOfTheServiceOverTheFoundOrder = 0.27
	toleranceBelowTheAcceptedMeanGain                   = 0.02
)

type documentsOrdering interface {
	OrderedDocumentsOf(answers queryanswers.AnsweredQuery) []queryanswers.FoundDocument
}

func TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries(t *testing.T) {
	t.Parallel()

	judged := judgedQueriesRecorded(t)
	gainOfTheOrderingOfTheService := gainPerJudgedQueryOf(
		orderingOfTheServiceFrom(documentrelevance.DefaultScoreWeights()), judged,
	)
	acceptedGain := acceptedGainPerJudgedQueryInTheFile(t, acceptedGainFile)

	reportTheGainOfEachJudgedQuery(t, judged, gainOfTheOrderingOfTheService)
	failIfTheLiftOverTheFoundOrderFallsShort(t, judged)
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

type orderingInTheFoundOrder struct{}

func (orderingInTheFoundOrder) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	return answers.FoundDocuments
}

func reportTheGainOfEachJudgedQuery(
	t *testing.T, judged []judgedQuery, gainOfTheOrderingOfTheService gainPerJudgedQuery,
) {
	t.Helper()

	documentRelevance := documentrelevance.New(documentrelevance.DefaultScoreWeights())
	relevanceOrdering := relevance.New(documentRelevance)
	amountOfSpamDocumentsInTheFirstTen := 0
	for _, judgedQuery := range judged {
		orderedDocuments := hostdiscount.New(documentRelevance).
			OrderedDocumentsOf(judgedQuery.answers)
		amountOfSpamDocumentsInTheFirstTen += judgedQuery.gradedDocuments.
			amountOfSpamDocumentsAmongTheFirstOf(orderedDocuments)
		t.Logf(
			"%q: host discount %.4f, relevance %.4f, found order %.4f, %d ungraded documents "+
				"dropped, %d spam documents in the first ten",
			judgedQuery.query,
			gainOfTheOrderingOfTheService[judgedQuery.query],
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
				relevanceOrdering.OrderedDocumentsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
				orderingInTheFoundOrder{}.OrderedDocumentsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.amountOfUngradedDocumentsAmong(orderedDocuments),
			judgedQuery.gradedDocuments.amountOfSpamDocumentsAmongTheFirstOf(orderedDocuments),
		)
	}
	t.Logf(
		"the mean over %d judged queries: host discount %.4f, relevance %.4f, found order %.4f",
		len(judged),
		meanNormalizedGainDiscountedPerHostOf(hostdiscount.New(documentRelevance), judged),
		meanNormalizedGainDiscountedPerHostOf(relevanceOrdering, judged),
		meanNormalizedGainDiscountedPerHostOf(orderingInTheFoundOrder{}, judged),
	)
	t.Logf(
		"the ordering of the service puts %d spam documents in the first ten over %d judged "+
			"queries",
		amountOfSpamDocumentsInTheFirstTen,
		len(judged),
	)
	t.Logf(
		"the ordering of the service reaches no gain on %d of %d judged queries",
		gainOfTheOrderingOfTheService.amountOfQueriesWithoutGain(),
		len(judged),
	)
}

func meanNormalizedGainDiscountedPerHostOf(
	ordering documentsOrdering, judged []judgedQuery,
) float64 {
	sumOfNormalizedGains := 0.0
	for _, judgedQuery := range judged {
		sumOfNormalizedGains += judgedQuery.gradedDocuments.normalizedGainDiscountedPerHostOf(
			ordering.OrderedDocumentsOf(judgedQuery.answers),
		)
	}

	return sumOfNormalizedGains / float64(len(judged))
}

func failIfTheLiftOverTheFoundOrderFallsShort(t *testing.T, judged []judgedQuery) {
	t.Helper()

	meanGainOfTheOrderingOfTheService := meanNormalizedGainDiscountedPerHostOf(
		orderingOfTheServiceFrom(documentrelevance.DefaultScoreWeights()), judged,
	)
	meanGainOfTheFoundOrder := meanNormalizedGainDiscountedPerHostOf(
		orderingInTheFoundOrder{}, judged,
	)
	if meanGainOfTheOrderingOfTheService-meanGainOfTheFoundOrder >=
		leastLiftOfTheOrderingOfTheServiceOverTheFoundOrder {
		return
	}
	t.Errorf(
		"the ordering of the service lifts the mean gain from %.4f to %.4f over %d judged "+
			"queries, want a lift of at least %.2f",
		meanGainOfTheFoundOrder,
		meanGainOfTheOrderingOfTheService,
		len(judged),
		leastLiftOfTheOrderingOfTheServiceOverTheFoundOrder,
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
