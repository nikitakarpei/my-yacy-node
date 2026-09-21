package judgedqueries_test

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"slices"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
)

const (
	tuningSwitch                         = "YACYDHTSEARCH_TUNE_RELEVANCE_WEIGHTS"
	amountOfWeightsOfTheRelevanceWeights = 5
	roundingOfTheWeightRatios            = 1e6
)

var (
	weightValuesOfTheGridOfTheTitleScore          = []float64{0, 3, 6, 10, 15}
	weightValuesOfTheGridOfTheTextScore           = []float64{0, 0.5, 1, 3, 6, 10}
	weightValuesOfTheGridOfThePhraseScore         = []float64{0, 0.25, 0.5, 1, 3}
	weightValuesOfTheGridOfTheNamedSiteEntryScore = []float64{0, 5, 15}
	weightValuesOfTheGridOfTheLinkSparsityPenalty = []float64{0, 1.5, 3}

	weightValuesOfTheGridOfEachScore = [amountOfWeightsOfTheRelevanceWeights][]float64{
		weightValuesOfTheGridOfTheTitleScore,
		weightValuesOfTheGridOfTheTextScore,
		weightValuesOfTheGridOfThePhraseScore,
		weightValuesOfTheGridOfTheNamedSiteEntryScore,
		weightValuesOfTheGridOfTheLinkSparsityPenalty,
	}
)

func TestTuneTheRelevanceWeights(t *testing.T) {
	if os.Getenv(tuningSwitch) == "" {
		t.Skipf("set %s to tune the relevance weights of the relevance ordering", tuningSwitch)
	}

	judged := judgedQueriesRecorded(t)
	reportTheBestRelevanceWeightsOverEveryJudgedQuery(t, judged)
	reportTheRelevanceWeightsTunedOnOneHalfAndMeasuredOnTheOther(t, judged)
}

func reportTheBestRelevanceWeightsOverEveryJudgedQuery(t *testing.T, judged []judgedQuery) {
	t.Helper()

	bestRelevanceWeights := bestRelevanceWeightsOver(judged)
	t.Logf(
		"over all %d judged queries the best of %d score weight vectors is %s and reaches the "+
			"mean gain %.4f; the default relevance weights reach %.4f",
		len(judged),
		len(relevanceWeightsOfTheGrid()),
		spelledRelevanceWeightsOf(bestRelevanceWeights),
		meanGainOfTheRelevanceWeights(bestRelevanceWeights, judged),
		meanGainOfTheRelevanceWeights(documentrelevance.DefaultRelevanceWeights(), judged),
	)
}

func reportTheRelevanceWeightsTunedOnOneHalfAndMeasuredOnTheOther(
	t *testing.T, judged []judgedQuery,
) {
	t.Helper()

	judgedQueriesPerHalf := judgedQueriesOfEachHalf(judged)
	for half, judgedQueriesOfTheHalf := range judgedQueriesPerHalf {
		heldOut := judgedQueriesPerHalf[len(judgedQueriesPerHalf)-1-half]
		tunedRelevanceWeights := bestRelevanceWeightsOver(judgedQueriesOfTheHalf)
		t.Logf(
			"tuned on half %d of %d queries, %s reaches the mean gain %.4f there and %.4f over "+
				"the held out %d queries, where the default relevance weights reach %.4f",
			half,
			len(judgedQueriesOfTheHalf),
			spelledRelevanceWeightsOf(tunedRelevanceWeights),
			meanGainOfTheRelevanceWeights(tunedRelevanceWeights, judgedQueriesOfTheHalf),
			meanGainOfTheRelevanceWeights(tunedRelevanceWeights, heldOut),
			len(heldOut),
			meanGainOfTheRelevanceWeights(documentrelevance.DefaultRelevanceWeights(), heldOut),
		)
	}
}

func judgedQueriesOfEachHalf(judged []judgedQuery) [2][]judgedQuery {
	inTheOrderOfTheGroups := slices.Clone(judged)
	slices.SortFunc(inTheOrderOfTheGroups, func(one, other judgedQuery) int {
		return placeAmongTheJudgedQueriesOf(one.query) -
			placeAmongTheJudgedQueriesOf(other.query)
	})

	var judgedQueriesPerHalf [2][]judgedQuery
	for place, judgedQueryOfTheGroups := range inTheOrderOfTheGroups {
		half := place % len(judgedQueriesPerHalf)
		judgedQueriesPerHalf[half] = append(judgedQueriesPerHalf[half], judgedQueryOfTheGroups)
	}

	return judgedQueriesPerHalf
}

func placeAmongTheJudgedQueriesOf(query string) int {
	return slices.Index(judgedQueries, query)
}

func bestRelevanceWeightsOver(judged []judgedQuery) documentrelevance.RelevanceWeights {
	grid := relevanceWeightsOfTheGrid()
	meanGainOfEachRelevanceWeights := meanGainOfEachOf(grid, judged)

	bestRelevanceWeights := documentrelevance.DefaultRelevanceWeights()
	bestMeanGain := meanGainOfTheRelevanceWeights(bestRelevanceWeights, judged)
	for place, meanGain := range meanGainOfEachRelevanceWeights {
		if meanGain <= bestMeanGain {
			continue
		}
		bestRelevanceWeights, bestMeanGain = grid[place], meanGain
	}

	return bestRelevanceWeights
}

func meanGainOfEachOf(
	grid []documentrelevance.RelevanceWeights, judged []judgedQuery,
) []float64 {
	meanGainOfEachRelevanceWeights := make([]float64, len(grid))
	amountOfWorkers := runtime.GOMAXPROCS(0)
	var measuringWorkers sync.WaitGroup
	for worker := range amountOfWorkers {
		measuringWorkers.Add(1)
		go func() {
			defer measuringWorkers.Done()
			for place := worker; place < len(grid); place += amountOfWorkers {
				meanGainOfEachRelevanceWeights[place] = meanGainOfTheRelevanceWeights(
					grid[place], judged,
				)
			}
		}()
	}
	measuringWorkers.Wait()

	return meanGainOfEachRelevanceWeights
}

func meanGainOfTheRelevanceWeights(
	relevanceWeights documentrelevance.RelevanceWeights, judged []judgedQuery,
) float64 {
	return meanNormalizedGainDiscountedPerSiteOf(orderingOfTheServiceFrom(relevanceWeights), judged)
}

func relevanceWeightsOfTheGrid() []documentrelevance.RelevanceWeights {
	return relevanceWeightsOfDistinctWeightRatiosAmong(relevanceWeightsOfEveryWeightCombination())
}

func relevanceWeightsOfEveryWeightCombination() []documentrelevance.RelevanceWeights {
	combinations := []documentrelevance.RelevanceWeights{{}}
	for score := range amountOfWeightsOfTheRelevanceWeights {
		combinations = combinationsOfEveryValueOfTheWeight(combinations, score)
	}

	return combinations
}

func combinationsOfEveryValueOfTheWeight(
	combinations []documentrelevance.RelevanceWeights, score int,
) []documentrelevance.RelevanceWeights {
	weightValues := weightValuesOfTheGridOfEachScore[score]
	widened := make([]documentrelevance.RelevanceWeights, 0, len(combinations)*len(weightValues))
	for _, relevanceWeights := range combinations {
		for _, weightValue := range weightValues {
			*weightOfEachScoreIn(&relevanceWeights)[score] = weightValue
			widened = append(widened, relevanceWeights)
		}
	}

	return widened
}

func weightOfEachScoreIn(
	relevanceWeights *documentrelevance.RelevanceWeights,
) [amountOfWeightsOfTheRelevanceWeights]*float64 {
	return [amountOfWeightsOfTheRelevanceWeights]*float64{
		&relevanceWeights.WeightOfTitleScore,
		&relevanceWeights.WeightOfTextScore,
		&relevanceWeights.WeightOfPhraseScore,
		&relevanceWeights.WeightOfNamedSiteEntryScore,
		&relevanceWeights.WeightOfLinkSparsityPenalty,
	}
}

func relevanceWeightsOfDistinctWeightRatiosAmong(
	combinations []documentrelevance.RelevanceWeights,
) []documentrelevance.RelevanceWeights {
	ofDistinctWeightRatios := make([]documentrelevance.RelevanceWeights, 0, len(combinations))
	alreadyTakenWeightRatios := map[[amountOfWeightsOfTheRelevanceWeights]float64]struct{}{}
	for _, relevanceWeights := range combinations {
		weightRatios := weightRatiosOf(relevanceWeights)
		if _, alreadyTaken := alreadyTakenWeightRatios[weightRatios]; alreadyTaken {
			continue
		}
		alreadyTakenWeightRatios[weightRatios] = struct{}{}
		ofDistinctWeightRatios = append(ofDistinctWeightRatios, relevanceWeights)
	}

	return ofDistinctWeightRatios
}

func weightRatiosOf(
	relevanceWeights documentrelevance.RelevanceWeights,
) [amountOfWeightsOfTheRelevanceWeights]float64 {
	var weights [amountOfWeightsOfTheRelevanceWeights]float64
	for place, weight := range weightOfEachScoreIn(&relevanceWeights) {
		weights[place] = *weight
	}
	highestWeight := slices.Max(weights[:])
	if highestWeight == 0 {
		return weights
	}
	for place, weight := range weights {
		weights[place] = math.Round(
			weight / highestWeight * roundingOfTheWeightRatios,
		)
	}

	return weights
}

func spelledRelevanceWeightsOf(relevanceWeights documentrelevance.RelevanceWeights) string {
	return fmt.Sprintf(
		"title %.2f, text %.2f, phrase %.2f, named site entry %.2f, link sparsity penalty %.2f",
		relevanceWeights.WeightOfTitleScore,
		relevanceWeights.WeightOfTextScore,
		relevanceWeights.WeightOfPhraseScore,
		relevanceWeights.WeightOfNamedSiteEntryScore,
		relevanceWeights.WeightOfLinkSparsityPenalty,
	)
}
