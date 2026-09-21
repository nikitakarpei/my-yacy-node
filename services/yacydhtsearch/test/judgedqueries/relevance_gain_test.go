package judgedqueries_test

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/sitediscount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

const (
	leastLiftOfTheOrderingOfTheServiceOverTheFoundOrder = 0.45
	toleranceBelowTheAcceptedMeanGain                   = 0.01
)

type documentsOrdering interface {
	OrderedDocumentsOf(answers queryanswers.AnsweredQuery) []queryanswers.FoundDocument
}

func TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries(t *testing.T) {
	t.Parallel()

	judged := judgedQueriesRecorded(t)
	orderingOfTheService := orderingOfTheServiceFrom(documentrelevance.DefaultRelevanceWeights())
	gainOfTheOrderingOfTheService := gainPerJudgedQueryOf(orderingOfTheService, judged)
	acceptedGain := acceptedGainPerJudgedQueryInTheFile(t, acceptedGainFile)
	judgedQueriesOfAnOrder := judgedQueriesOfSeveralRelevantDocuments(judged)

	reportTheGainOfEachJudgedQuery(t, judged, gainOfTheOrderingOfTheService)
	reportTheMeanGainOfEachOrdering(t, judgedQueriesOfAnOrder)
	reportTheRelevantDocumentsHeldInTheFirstTen(
		t, judgedQueriesOfOneRelevantDocument(judged), gainOfTheOrderingOfTheService,
	)
	failIfTheLiftOverTheFoundOrderFallsShort(t, judgedQueriesOfAnOrder)
	failIfTheMeanGainFallsBelowTheAcceptedGain(
		t, gainPerJudgedQueryOf(orderingOfTheService, judgedQueriesOfAnOrder), acceptedGain,
	)
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
	relevanceWeights documentrelevance.RelevanceWeights,
) sitediscount.Ordering {
	return sitediscount.New(documentrelevance.RelevanceScorerWeighedBy(relevanceWeights))
}

type orderingInTheFoundOrder struct{}

func (orderingInTheFoundOrder) OrderedDocumentsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.FoundDocument {
	return answers.FoundDocuments
}

func judgedQueriesOfSeveralRelevantDocuments(judged []judgedQuery) []judgedQuery {
	ofSeveralRelevantDocuments := make([]judgedQuery, 0, len(judged))
	for _, judgedQuery := range judged {
		if !judgedQuery.gradedDocuments.holdAnOrderOfRelevantDocuments() {
			continue
		}
		ofSeveralRelevantDocuments = append(ofSeveralRelevantDocuments, judgedQuery)
	}

	return ofSeveralRelevantDocuments
}

func judgedQueriesOfOneRelevantDocument(judged []judgedQuery) []judgedQuery {
	ofOneRelevantDocument := make([]judgedQuery, 0, len(judged))
	for _, judgedQuery := range judged {
		if judgedQuery.gradedDocuments.holdAnOrderOfRelevantDocuments() {
			continue
		}
		ofOneRelevantDocument = append(ofOneRelevantDocument, judgedQuery)
	}

	return ofOneRelevantDocument
}

func reportTheGainOfEachJudgedQuery(
	t *testing.T, judged []judgedQuery, gainOfTheOrderingOfTheService gainPerJudgedQuery,
) {
	t.Helper()

	relevanceScorer := documentrelevance.RelevanceScorerWeighedBy(
		documentrelevance.DefaultRelevanceWeights(),
	)
	relevanceOrdering := relevance.New(relevanceScorer)
	amountOfSpamDocumentsInTheFirstTen := 0
	for _, judgedQuery := range judged {
		orderedDocuments := sitediscount.New(relevanceScorer).
			OrderedDocumentsOf(judgedQuery.answers)
		amountOfSpamDocumentsInTheFirstTen += judgedQuery.gradedDocuments.
			amountOfSpamDocumentsAmongTheFirstOf(orderedDocuments)
		t.Logf(
			"%q: site discount %.4f, relevance %.4f, found order %.4f, %d ungraded documents "+
				"dropped, %d spam documents in the first ten",
			judgedQuery.query,
			gainOfTheOrderingOfTheService[judgedQuery.query],
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerSiteOf(
				relevanceOrdering.OrderedDocumentsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.normalizedGainDiscountedPerSiteOf(
				orderingInTheFoundOrder{}.OrderedDocumentsOf(judgedQuery.answers),
			),
			judgedQuery.gradedDocuments.amountOfUngradedDocumentsAmong(orderedDocuments),
			judgedQuery.gradedDocuments.amountOfSpamDocumentsAmongTheFirstOf(orderedDocuments),
		)
	}
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

func reportTheRelevantDocumentsHeldInTheFirstTen(
	t *testing.T,
	judgedQueriesOfOneRelevantDocument []judgedQuery,
	gainOfTheOrderingOfTheService gainPerJudgedQuery,
) {
	t.Helper()

	heldInTheFirstTen := 0
	for _, judgedQuery := range judgedQueriesOfOneRelevantDocument {
		if gainOfTheOrderingOfTheService[judgedQuery.query] > 0 {
			heldInTheFirstTen++
		}
	}
	t.Logf(
		"the ordering of the service holds the one relevant document of %d of %d judged "+
			"queries in the first ten, whose gain measures its place and not an order",
		heldInTheFirstTen,
		len(judgedQueriesOfOneRelevantDocument),
	)
}

func reportTheMeanGainOfEachOrdering(t *testing.T, judgedQueriesOfAnOrder []judgedQuery) {
	t.Helper()

	relevanceScorer := documentrelevance.RelevanceScorerWeighedBy(
		documentrelevance.DefaultRelevanceWeights(),
	)
	t.Logf(
		"the mean over the %d judged queries of several relevant documents: site discount "+
			"%.4f, relevance %.4f, found order %.4f",
		len(judgedQueriesOfAnOrder),
		meanNormalizedGainDiscountedPerSiteOf(
			sitediscount.New(relevanceScorer), judgedQueriesOfAnOrder,
		),
		meanNormalizedGainDiscountedPerSiteOf(
			relevance.New(relevanceScorer), judgedQueriesOfAnOrder,
		),
		meanNormalizedGainDiscountedPerSiteOf(orderingInTheFoundOrder{}, judgedQueriesOfAnOrder),
	)
}

func meanNormalizedGainDiscountedPerSiteOf(
	ordering documentsOrdering, judged []judgedQuery,
) float64 {
	sumOfNormalizedGains := 0.0
	for _, judgedQuery := range judged {
		sumOfNormalizedGains += judgedQuery.gradedDocuments.normalizedGainDiscountedPerSiteOf(
			ordering.OrderedDocumentsOf(judgedQuery.answers),
		)
	}

	return sumOfNormalizedGains / float64(len(judged))
}

func failIfTheLiftOverTheFoundOrderFallsShort(t *testing.T, judged []judgedQuery) {
	t.Helper()

	meanGainOfTheOrderingOfTheService := meanNormalizedGainDiscountedPerSiteOf(
		orderingOfTheServiceFrom(documentrelevance.DefaultRelevanceWeights()), judged,
	)
	meanGainOfTheFoundOrder := meanNormalizedGainDiscountedPerSiteOf(
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
