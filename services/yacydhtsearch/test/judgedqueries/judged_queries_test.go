package judgedqueries_test

import (
	"runtime"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
)

type judgedQuery struct {
	query           string
	findings        queryfindings.Findings
	gradedDocuments gradedDocuments
}

type judgedQueries []judgedQuery

type documentsOrdering interface {
	OrderedDocumentsOf(findings queryfindings.Findings) []queryfindings.FoundDocument
}

func judgedQueriesRecorded(t *testing.T) judgedQueries {
	t.Helper()

	judgmentsFiles := queryJudgmentsFiles(t)
	findingsFiles := recordedFindingsFiles(t)
	if len(judgmentsFiles) != len(findingsFiles) {
		t.Fatalf("%s judges %d queries, %s records %d queries",
			queryJudgmentsDirectory, len(judgmentsFiles),
			recordedFindingsDirectory, len(findingsFiles))
	}
	var queries judgedQueries
	for _, judgmentsFile := range judgmentsFiles {
		judgments := queryJudgmentsAt(t, judgmentsFile)
		gradedDocuments := judgments.gradedDocuments()
		if !gradedDocuments.holdARelevantDocument() {
			continue
		}
		recorded := recordedFindingsAt(t, recordedFindingsFileOf(judgments.Query))
		queries = append(queries, judgedQuery{
			query:           recorded.Query,
			findings:        recorded.findings(),
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
	return queries.orderedBy(ordering).gainPerQuery()
}

func (queries judgedQueries) orderedBy(ordering documentsOrdering) orderedQueries {
	ordered := make(orderedQueries, 0, len(queries))
	for _, judgedQuery := range queries {
		ordered = append(ordered, orderedQuery{
			judgedQuery:      judgedQuery,
			orderedDocuments: ordering.OrderedDocumentsOf(judgedQuery.findings),
		})
	}

	return ordered
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
