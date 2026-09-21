package judgedqueries_test

import (
	"encoding/json"
	"os"
	"testing"
)

const acceptedGainFile = "testdata/accepted-gain-per-judged-query.json"

type gainPerQuery map[string]float64

func acceptedGainPerQuery(t *testing.T) gainPerQuery {
	t.Helper()

	content, err := os.ReadFile(acceptedGainFile)
	if err != nil {
		t.Fatalf("read %s: %v", acceptedGainFile, err)
	}
	var acceptedGain gainPerQuery
	if err := json.Unmarshal(content, &acceptedGain); err != nil {
		t.Fatalf("read %s: %v", acceptedGainFile, err)
	}

	return acceptedGain
}

func (gain gainPerQuery) meanGain() float64 {
	if len(gain) == 0 {
		return 0
	}
	sumOfGains := 0.0
	for _, gainOfQuery := range gain {
		sumOfGains += gainOfQuery
	}

	return sumOfGains / float64(len(gain))
}

func (gain gainPerQuery) meanGainSharedWith(other gainPerQuery) float64 {
	sumOfGains := 0.0
	amountOfSharedQueries := 0
	for query, gainOfQuery := range gain {
		if _, shared := other[query]; !shared {
			continue
		}
		sumOfGains += gainOfQuery
		amountOfSharedQueries++
	}
	if amountOfSharedQueries == 0 {
		return 0
	}

	return sumOfGains / float64(amountOfSharedQueries)
}

func (gain gainPerQuery) amountOfQueriesWithoutGain() int {
	amountOfQueriesWithoutGain := 0
	for _, gainOfQuery := range gain {
		if gainOfQuery > 0 {
			continue
		}
		amountOfQueriesWithoutGain++
	}

	return amountOfQueriesWithoutGain
}
