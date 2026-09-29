package spammodel_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/featurehashing"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
)

func TestAScoreIsTheLogisticOfTheInterceptHashedAndStandardisedMetaMargins(t *testing.T) {
	model := spammodel.Model{
		Intercept: 1,
		HashedWeights: map[spamfeatures.Family]spammodel.HashedWeights{
			spamfeatures.Text:  spammodel.HashedWeightsFrom([]int32{7}, []float64{2}),
			spamfeatures.Links: spammodel.HashedWeightsFrom([]int32{3}, []float64{-4}),
		},
		MetaWeights:    []float64{3},
		MetaMeans:      []float64{1},
		MetaDeviations: []float64{2},
	}
	row := spamfeatures.Row{
		HashedEntries: map[spamfeatures.Family][]featurehashing.Entry{
			spamfeatures.Text:  {{Index: 7, Value: 0.5}, {Index: 8, Value: 0.5}},
			spamfeatures.Links: {{Index: 3, Value: 0.25}},
		},
		MetaValues: []float64{5},
	}

	score := model.ScoreOf(row)

	if want := 1 / (1 + math.Exp(-7)); math.Abs(score-want) > 1e-12 {
		t.Errorf("ScoreOf = %v, want %v", score, want)
	}
}

func TestAnOverwhelmingMarginScoresAtTheBounds(t *testing.T) {
	scoreOfIntercept := func(intercept float64) float64 {
		return spammodel.Model{Intercept: intercept}.ScoreOf(spamfeatures.Row{})
	}

	if low, high := scoreOfIntercept(-1000), scoreOfIntercept(1000); low != 0 || high != 1 {
		t.Errorf("scores = %v and %v, want 0 and 1", low, high)
	}
}
