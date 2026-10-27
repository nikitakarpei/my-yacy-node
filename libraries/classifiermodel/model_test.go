package classifiermodel_test

import (
	"math"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/featurehashing"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/pagefeatures"
)

func TestAScoreIsTheLogisticOfTheInterceptHashedAndStandardisedMetaMargins(t *testing.T) {
	model := classifiermodel.Model{
		Intercept: 1,
		HashedWeights: map[pagefeatures.Family]classifiermodel.HashedWeights{
			pagefeatures.Text:  classifiermodel.HashedWeightsFrom([]int32{7}, []float64{2}),
			pagefeatures.Links: classifiermodel.HashedWeightsFrom([]int32{3}, []float64{-4}),
		},
		MetaWeights:    []float64{3},
		MetaMeans:      []float64{1},
		MetaDeviations: []float64{2},
	}
	row := pagefeatures.Row{
		HashedEntries: map[pagefeatures.Family][]featurehashing.Entry{
			pagefeatures.Text:  {{Index: 7, Value: 0.5}, {Index: 8, Value: 0.5}},
			pagefeatures.Links: {{Index: 3, Value: 0.25}},
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
		return classifiermodel.Model{Intercept: intercept}.ScoreOf(pagefeatures.Row{})
	}

	if low, high := scoreOfIntercept(-1000), scoreOfIntercept(1000); low != 0 || high != 1 {
		t.Errorf("scores = %v and %v, want 0 and 1", low, high)
	}
}
