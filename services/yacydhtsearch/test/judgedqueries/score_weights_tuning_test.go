package judgedqueries_test

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"slices"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
)

const (
	tuningSwitch                     = "YACYDHTSEARCH_TUNE_RELEVANCE_WEIGHTS"
	amountOfWeightsOfTheScoreWeights = 5
	roundingOfTheWeightRatios        = 1e6
)

var weightValuesOfTheGrid = []float64{0, 0.25, 0.5, 1, 1.5, 2, 3}

func TestTuneTheScoreWeightsOfTheRelevanceOrdering(t *testing.T) {
	if os.Getenv(tuningSwitch) == "" {
		t.Skipf("set %s to tune the score weights of the relevance ordering", tuningSwitch)
	}

	judged := judgedQueriesRecorded(t)
	reportTheBestScoreWeightsOverEveryJudgedQuery(t, judged)
	reportTheScoreWeightsTunedOnOneHalfAndMeasuredOnTheOther(t, judged)
}

func reportTheBestScoreWeightsOverEveryJudgedQuery(t *testing.T, judged []judgedQuery) {
	t.Helper()

	bestScoreWeights := bestScoreWeightsOver(judged)
	t.Logf(
		"over all %d judged queries the best of %d score weight vectors is %s and reaches the "+
			"mean gain %.4f; the default score weights reach %.4f",
		len(judged),
		len(scoreWeightsOfTheGrid()),
		spelledScoreWeightsOf(bestScoreWeights),
		meanGainOfTheScoreWeights(bestScoreWeights, judged),
		meanGainOfTheScoreWeights(relevance.DefaultScoreWeights(), judged),
	)
}

func reportTheScoreWeightsTunedOnOneHalfAndMeasuredOnTheOther(
	t *testing.T, judged []judgedQuery,
) {
	t.Helper()

	judgedQueriesPerHalf := judgedQueriesOfEachHalf(judged)
	for half, judgedQueriesOfTheHalf := range judgedQueriesPerHalf {
		heldOut := judgedQueriesPerHalf[len(judgedQueriesPerHalf)-1-half]
		tunedScoreWeights := bestScoreWeightsOver(judgedQueriesOfTheHalf)
		t.Logf(
			"tuned on half %d of %d queries, %s reaches the mean gain %.4f there and %.4f over "+
				"the held out %d queries, where the default score weights reach %.4f",
			half,
			len(judgedQueriesOfTheHalf),
			spelledScoreWeightsOf(tunedScoreWeights),
			meanGainOfTheScoreWeights(tunedScoreWeights, judgedQueriesOfTheHalf),
			meanGainOfTheScoreWeights(tunedScoreWeights, heldOut),
			len(heldOut),
			meanGainOfTheScoreWeights(relevance.DefaultScoreWeights(), heldOut),
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

func bestScoreWeightsOver(judged []judgedQuery) relevance.ScoreWeights {
	grid := scoreWeightsOfTheGrid()
	meanGainOfEachScoreWeights := meanGainOfEachOf(grid, judged)

	bestScoreWeights := relevance.DefaultScoreWeights()
	bestMeanGain := meanGainOfTheScoreWeights(bestScoreWeights, judged)
	for place, meanGain := range meanGainOfEachScoreWeights {
		if meanGain <= bestMeanGain {
			continue
		}
		bestScoreWeights, bestMeanGain = grid[place], meanGain
	}

	return bestScoreWeights
}

func meanGainOfEachOf(
	grid []relevance.ScoreWeights, judged []judgedQuery,
) []float64 {
	meanGainOfEachScoreWeights := make([]float64, len(grid))
	amountOfWorkers := runtime.GOMAXPROCS(0)
	var measuringWorkers sync.WaitGroup
	for worker := range amountOfWorkers {
		measuringWorkers.Add(1)
		go func() {
			defer measuringWorkers.Done()
			for place := worker; place < len(grid); place += amountOfWorkers {
				meanGainOfEachScoreWeights[place] = meanGainOfTheScoreWeights(
					grid[place], judged,
				)
			}
		}()
	}
	measuringWorkers.Wait()

	return meanGainOfEachScoreWeights
}

func meanGainOfTheScoreWeights(
	scoreWeights relevance.ScoreWeights, judged []judgedQuery,
) float64 {
	return meanNormalizedGainDiscountedPerHostOf(orderingOfTheServiceFrom(scoreWeights), judged)
}

func scoreWeightsOfTheGrid() []relevance.ScoreWeights {
	return scoreWeightsOfDistinctWeightRatiosAmong(scoreWeightsOfEveryWeightCombination())
}

func scoreWeightsOfEveryWeightCombination() []relevance.ScoreWeights {
	amountOfVectors := 1
	for range amountOfWeightsOfTheScoreWeights {
		amountOfVectors *= len(weightValuesOfTheGrid)
	}

	combinations := make([]relevance.ScoreWeights, 0, amountOfVectors)
	for _, weightOfThePlaceScore := range weightValuesOfTheGrid {
		for _, weightOfTheTitleScore := range weightValuesOfTheGrid {
			for _, weightOfTheTextScore := range weightValuesOfTheGrid {
				for _, weightOfTheAddressScore := range weightValuesOfTheGrid {
					for _, weightOfThePhraseScore := range weightValuesOfTheGrid {
						combinations = append(combinations, relevance.ScoreWeights{
							WeightOfThePlaceScore:   weightOfThePlaceScore,
							WeightOfTheTitleScore:   weightOfTheTitleScore,
							WeightOfTheTextScore:    weightOfTheTextScore,
							WeightOfTheAddressScore: weightOfTheAddressScore,
							WeightOfThePhraseScore:  weightOfThePhraseScore,
						})
					}
				}
			}
		}
	}

	return combinations
}

func scoreWeightsOfDistinctWeightRatiosAmong(
	combinations []relevance.ScoreWeights,
) []relevance.ScoreWeights {
	ofDistinctWeightRatios := make([]relevance.ScoreWeights, 0, len(combinations))
	alreadyTakenWeightRatios := map[[amountOfWeightsOfTheScoreWeights]float64]struct{}{}
	for _, scoreWeights := range combinations {
		weightRatios := weightRatiosOf(scoreWeights)
		if _, alreadyTaken := alreadyTakenWeightRatios[weightRatios]; alreadyTaken {
			continue
		}
		alreadyTakenWeightRatios[weightRatios] = struct{}{}
		ofDistinctWeightRatios = append(ofDistinctWeightRatios, scoreWeights)
	}

	return ofDistinctWeightRatios
}

func weightRatiosOf(
	scoreWeights relevance.ScoreWeights,
) [amountOfWeightsOfTheScoreWeights]float64 {
	weights := [amountOfWeightsOfTheScoreWeights]float64{
		scoreWeights.WeightOfThePlaceScore,
		scoreWeights.WeightOfTheTitleScore,
		scoreWeights.WeightOfTheTextScore,
		scoreWeights.WeightOfTheAddressScore,
		scoreWeights.WeightOfThePhraseScore,
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

func spelledScoreWeightsOf(scoreWeights relevance.ScoreWeights) string {
	return fmt.Sprintf(
		"place %.2f, title %.2f, text %.2f, address %.2f, phrase %.2f",
		scoreWeights.WeightOfThePlaceScore,
		scoreWeights.WeightOfTheTitleScore,
		scoreWeights.WeightOfTheTextScore,
		scoreWeights.WeightOfTheAddressScore,
		scoreWeights.WeightOfThePhraseScore,
	)
}
