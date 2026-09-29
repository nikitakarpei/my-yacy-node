package spammodel_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
)

func TestAWeightIsFoundByItsIndexWhateverTheOrderGiven(t *testing.T) {
	weights := spammodel.HashedWeightsFrom([]int32{9, 2, 5}, []float64{0.9, 0.2, 0.5})

	if weight := weights.WeightOf(5); weight != 0.5 {
		t.Errorf("WeightOf(5) = %v, want 0.5", weight)
	}
}

func TestAnIndexWithoutAWeightWeighsNothing(t *testing.T) {
	weights := spammodel.HashedWeightsFrom([]int32{9, 2}, []float64{0.9, 0.2})

	if weight := weights.WeightOf(4); weight != 0 {
		t.Errorf("WeightOf(4) = %v, want 0", weight)
	}
}

func TestEntriesComeInIndexOrder(t *testing.T) {
	weights := spammodel.HashedWeightsFrom([]int32{9, 2, 5}, []float64{0.9, 0.2, 0.5})

	var indices []int32
	for index := range weights.Entries() {
		indices = append(indices, index)
		if index == 5 {
			break
		}
	}

	if want := []int32{2, 5}; !slices.Equal(indices, want) {
		t.Errorf("indices = %v, want %v", indices, want)
	}
	if weightOfIndex := maps.Collect(weights.Entries()); weightOfIndex[9] != 0.9 {
		t.Errorf("weight of 9 = %v, want 0.9", weightOfIndex[9])
	}
}
