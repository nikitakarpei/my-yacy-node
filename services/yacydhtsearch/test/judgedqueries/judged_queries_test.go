package judgedqueries_test

import (
	"runtime"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

type judgedQuery struct {
	query           string
	answers         queryanswers.AnsweredQuery
	gradedDocuments gradedDocuments
}

type judgedQueries []judgedQuery

type documentsOrdering interface {
	OrderedDocumentsOf(answers queryanswers.AnsweredQuery) []queryanswers.FoundDocument
}

func judgedQueriesRecorded(t *testing.T) judgedQueries {
	t.Helper()

	judgmentsFiles := queryJudgmentsFiles(t)
	answersFiles := recordedAnswersFiles(t)
	if len(judgmentsFiles) != len(answersFiles) {
		t.Fatalf("%s judges %d queries, %s records %d queries",
			queryJudgmentsDirectory, len(judgmentsFiles),
			recordedAnswersDirectory, len(answersFiles))
	}
	var queries judgedQueries
	for _, judgmentsFile := range judgmentsFiles {
		judgments := queryJudgmentsAt(t, judgmentsFile)
		gradedDocuments := judgments.gradedDocuments()
		if !gradedDocuments.holdARelevantDocument() {
			continue
		}
		recorded := recordedAnswersAt(t, recordedAnswersFileOf(judgments.Query))
		queries = append(queries, judgedQuery{
			query:           recorded.Query,
			answers:         recorded.answers(),
			gradedDocuments: gradedDocuments,
		})
	}

	return queries
}

func (queries judgedQueries) ofSeveralRelevantDocuments() judgedQueries {
	queriesOfSeveralRelevantDocuments := make(judgedQueries, 0, len(queries))
	for _, judgedQuery := range queries {
		if !judgedQuery.gradedDocuments.holdSeveralRelevantDocuments() {
			continue
		}
		queriesOfSeveralRelevantDocuments = append(queriesOfSeveralRelevantDocuments, judgedQuery)
	}

	return queriesOfSeveralRelevantDocuments
}

func (queries judgedQueries) ofOneRelevantDocument() judgedQueries {
	queriesOfOneRelevantDocument := make(judgedQueries, 0, len(queries))
	for _, judgedQuery := range queries {
		if judgedQuery.gradedDocuments.holdSeveralRelevantDocuments() {
			continue
		}
		queriesOfOneRelevantDocument = append(queriesOfOneRelevantDocument, judgedQuery)
	}

	return queriesOfOneRelevantDocument
}

func (queries judgedQueries) gainPerQueryOf(ordering documentsOrdering) gainPerQuery {
	gain := make(gainPerQuery, len(queries))
	for _, judgedQuery := range queries {
		gain[judgedQuery.query] = judgedQuery.gradedDocuments.
			normalizedGainOf(ordering.OrderedDocumentsOf(judgedQuery.answers))
	}

	return gain
}

func (queries judgedQueries) meanGainOf(ordering documentsOrdering) float64 {
	return queries.gainPerQueryOf(ordering).meanGain()
}

func (queries judgedQueries) meanGainWeighedBy(
	relevanceWeights documentrelevance.RelevanceWeights,
) float64 {
	return queries.meanGainOf(serviceOrderingWeighedBy(relevanceWeights))
}

func (queries judgedQueries) meanGainOfEachIn(
	grid []documentrelevance.RelevanceWeights,
) []float64 {
	meanGainPerRelevanceWeights := make([]float64, len(grid))
	amountOfWorkers := runtime.GOMAXPROCS(0)
	var measuringWorkers sync.WaitGroup
	for worker := range amountOfWorkers {
		measuringWorkers.Add(1)
		go func() {
			defer measuringWorkers.Done()
			for place := worker; place < len(grid); place += amountOfWorkers {
				meanGainPerRelevanceWeights[place] = queries.meanGainWeighedBy(grid[place])
			}
		}()
	}
	measuringWorkers.Wait()

	return meanGainPerRelevanceWeights
}
