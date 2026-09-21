package judgedqueries_test

import (
	"fmt"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
)

const (
	tuningSwitch           = "YACYDHTSEARCH_TUNE_RELEVANCE_WEIGHTS"
	amountOfWeightedScores = 5
	weightRatioRounding    = 1e6
)

var (
	titleScoreWeightGrid          = []float64{0, 3, 6, 10, 15}
	textScoreWeightGrid           = []float64{0, 0.5, 1, 3, 6, 10}
	phraseScoreWeightGrid         = []float64{0, 0.25, 0.5, 1, 3}
	namedSiteEntryScoreWeightGrid = []float64{0, 5, 15}
	linkSparsityPenaltyWeightGrid = []float64{0, 1.5, 3}

	gridPerWeight = [amountOfWeightedScores][]float64{
		titleScoreWeightGrid,
		textScoreWeightGrid,
		phraseScoreWeightGrid,
		namedSiteEntryScoreWeightGrid,
		linkSparsityPenaltyWeightGrid,
	}
)

func TestTuneTheRelevanceWeights(t *testing.T) {
	if os.Getenv(tuningSwitch) == "" {
		t.Skipf("set %s to tune the relevance weights of the relevance ordering", tuningSwitch)
	}

	queries := judgedQueriesRecorded(t)
	reportBestRelevanceWeights(t, queries)
	reportRelevanceWeightsTunedOnEachHalf(t, queries)
}

func reportBestRelevanceWeights(t *testing.T, queries judgedQueries) {
	t.Helper()

	bestRelevanceWeights := bestRelevanceWeightsOver(queries)
	t.Logf(
		"over all %d judged queries the best of %d score weight vectors is %s and reaches the "+
			"mean gain %.4f; the default relevance weights reach %.4f",
		len(queries),
		len(relevanceWeightsGrid()),
		spellingOf(bestRelevanceWeights),
		queries.meanGainWeighedBy(bestRelevanceWeights),
		queries.meanGainWeighedBy(documentrelevance.DefaultRelevanceWeights()),
	)
}

func bestRelevanceWeightsOver(queries judgedQueries) documentrelevance.RelevanceWeights {
	grid := relevanceWeightsGrid()
	meanGainPerRelevanceWeights := queries.meanGainOfEachIn(grid)

	bestRelevanceWeights := documentrelevance.DefaultRelevanceWeights()
	bestMeanGain := queries.meanGainWeighedBy(bestRelevanceWeights)
	for place, meanGain := range meanGainPerRelevanceWeights {
		if meanGain <= bestMeanGain {
			continue
		}
		bestRelevanceWeights, bestMeanGain = grid[place], meanGain
	}

	return bestRelevanceWeights
}

func relevanceWeightsGrid() []documentrelevance.RelevanceWeights {
	return relevanceWeightsOfDistinctRatiosAmong(everyWeightCombination())
}

func everyWeightCombination() []documentrelevance.RelevanceWeights {
	combinations := []documentrelevance.RelevanceWeights{{}}
	for weight := range amountOfWeightedScores {
		combinations = combinationsWidenedByTheWeight(combinations, weight)
	}

	return combinations
}

func combinationsWidenedByTheWeight(
	combinations []documentrelevance.RelevanceWeights, weight int,
) []documentrelevance.RelevanceWeights {
	weightValues := gridPerWeight[weight]
	widened := make([]documentrelevance.RelevanceWeights, 0, len(combinations)*len(weightValues))
	for _, relevanceWeights := range combinations {
		for _, weightValue := range weightValues {
			weights := weightsOf(relevanceWeights)
			weights[weight] = weightValue
			widened = append(widened, relevanceWeightsOf(weights))
		}
	}

	return widened
}

func weightsOf(
	relevanceWeights documentrelevance.RelevanceWeights,
) [amountOfWeightedScores]float64 {
	return [amountOfWeightedScores]float64{
		relevanceWeights.WeightOfTitleScore,
		relevanceWeights.WeightOfTextScore,
		relevanceWeights.WeightOfPhraseScore,
		relevanceWeights.WeightOfNamedSiteEntryScore,
		relevanceWeights.WeightOfLinkSparsityPenalty,
	}
}

func relevanceWeightsOf(
	weights [amountOfWeightedScores]float64,
) documentrelevance.RelevanceWeights {
	return documentrelevance.RelevanceWeights{
		WeightOfTitleScore:          weights[0],
		WeightOfTextScore:           weights[1],
		WeightOfPhraseScore:         weights[2],
		WeightOfNamedSiteEntryScore: weights[3],
		WeightOfLinkSparsityPenalty: weights[4],
	}
}

func relevanceWeightsOfDistinctRatiosAmong(
	combinations []documentrelevance.RelevanceWeights,
) []documentrelevance.RelevanceWeights {
	relevanceWeightsOfDistinctRatios := make(
		[]documentrelevance.RelevanceWeights, 0, len(combinations),
	)
	alreadyTakenWeightRatios := map[[amountOfWeightedScores]float64]struct{}{}
	for _, relevanceWeights := range combinations {
		weightRatios := weightRatiosOf(relevanceWeights)
		if _, alreadyTaken := alreadyTakenWeightRatios[weightRatios]; alreadyTaken {
			continue
		}
		alreadyTakenWeightRatios[weightRatios] = struct{}{}
		relevanceWeightsOfDistinctRatios = append(
			relevanceWeightsOfDistinctRatios,
			relevanceWeights,
		)
	}

	return relevanceWeightsOfDistinctRatios
}

func weightRatiosOf(
	relevanceWeights documentrelevance.RelevanceWeights,
) [amountOfWeightedScores]float64 {
	weights := weightsOf(relevanceWeights)
	highestWeight := slices.Max(weights[:])
	if highestWeight == 0 {
		return weights
	}
	for weight, weightValue := range weights {
		weights[weight] = math.Round(weightValue / highestWeight * weightRatioRounding)
	}

	return weights
}

func spellingOf(relevanceWeights documentrelevance.RelevanceWeights) string {
	return fmt.Sprintf(
		"title %.2f, text %.2f, phrase %.2f, named site entry %.2f, link sparsity penalty %.2f",
		relevanceWeights.WeightOfTitleScore,
		relevanceWeights.WeightOfTextScore,
		relevanceWeights.WeightOfPhraseScore,
		relevanceWeights.WeightOfNamedSiteEntryScore,
		relevanceWeights.WeightOfLinkSparsityPenalty,
	)
}

func reportRelevanceWeightsTunedOnEachHalf(t *testing.T, queries judgedQueries) {
	t.Helper()

	halves := halvesOf(queries)
	for half, queriesOfTheHalf := range halves {
		heldOutQueries := halves[len(halves)-1-half]
		tunedRelevanceWeights := bestRelevanceWeightsOver(queriesOfTheHalf)
		t.Logf(
			"tuned on half %d of %d queries, %s reaches the mean gain %.4f there and %.4f over "+
				"the held out %d queries, where the default relevance weights reach %.4f",
			half,
			len(queriesOfTheHalf),
			spellingOf(tunedRelevanceWeights),
			queriesOfTheHalf.meanGainWeighedBy(tunedRelevanceWeights),
			heldOutQueries.meanGainWeighedBy(tunedRelevanceWeights),
			len(heldOutQueries),
			heldOutQueries.meanGainWeighedBy(documentrelevance.DefaultRelevanceWeights()),
		)
	}
}

func halvesOf(queries judgedQueries) [2]judgedQueries {
	queriesInTheRecordedOrder := slices.Clone(queries)
	slices.SortFunc(queriesInTheRecordedOrder, func(one, other judgedQuery) int {
		return slices.Index(recordedQueries, one.query) -
			slices.Index(recordedQueries, other.query)
	})

	var halves [2]judgedQueries
	for place, judgedQuery := range queriesInTheRecordedOrder {
		half := place % len(halves)
		halves[half] = append(halves[half], judgedQuery)
	}

	return halves
}
