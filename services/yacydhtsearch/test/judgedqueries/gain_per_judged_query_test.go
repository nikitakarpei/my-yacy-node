package judgedqueries_test

import (
	"encoding/json"
	"os"
	"testing"
)

const acceptedGainFile = "testdata/accepted-gain-per-judged-query.json"

type gainPerJudgedQuery map[string]float64

func gainPerJudgedQueryOf(
	ordering itemsOrdering, judged []judgedQuery,
) gainPerJudgedQuery {
	gainOfEachQuery := make(gainPerJudgedQuery, len(judged))
	for _, judgedQuery := range judged {
		gainOfEachQuery[judgedQuery.query] = judgedQuery.gradedDocuments.
			normalizedGainDiscountedPerHostOf(ordering.OrderedItemsOf(judgedQuery.answers))
	}

	return gainOfEachQuery
}

func (gainOfEachQuery gainPerJudgedQuery) meanGainOverTheQueriesIn(
	otherGainOfEachQuery gainPerJudgedQuery,
) float64 {
	sumOfGains := 0.0
	amountOfSharedQueries := 0
	for query, gain := range gainOfEachQuery {
		if _, shared := otherGainOfEachQuery[query]; !shared {
			continue
		}
		sumOfGains += gain
		amountOfSharedQueries++
	}
	if amountOfSharedQueries == 0 {
		return 0
	}

	return sumOfGains / float64(amountOfSharedQueries)
}

func (gainOfEachQuery gainPerJudgedQuery) amountOfQueriesWithoutGain() int {
	amountOfQueriesWithoutGain := 0
	for _, gain := range gainOfEachQuery {
		if gain > 0 {
			continue
		}
		amountOfQueriesWithoutGain++
	}

	return amountOfQueriesWithoutGain
}

func acceptedGainPerJudgedQueryInTheFile(t *testing.T, path string) gainPerJudgedQuery {
	t.Helper()

	content, err := os.ReadFile(path) //nolint:gosec // a fixture path of this test directory
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var acceptedGainOfEachQuery gainPerJudgedQuery
	if err := json.Unmarshal(content, &acceptedGainOfEachQuery); err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return acceptedGainOfEachQuery
}
