package spammodel

import (
	"iter"
	"slices"
	"sort"
)

type HashedWeights struct {
	indices []int32
	weights []float64
}

func HashedWeightsFrom(indices []int32, weights []float64) HashedWeights {
	sortedWeights := HashedWeights{indices: slices.Clone(indices), weights: slices.Clone(weights)}
	sort.Sort(indexOrder(sortedWeights))
	return sortedWeights
}

func (w HashedWeights) WeightOf(index int32) float64 {
	position, found := slices.BinarySearch(w.indices, index)
	if !found {
		return 0
	}
	return w.weights[position]
}

func (w HashedWeights) Entries() iter.Seq2[int32, float64] {
	return func(yield func(int32, float64) bool) {
		for position, index := range w.indices {
			if !yield(index, w.weights[position]) {
				return
			}
		}
	}
}

type indexOrder HashedWeights

func (o indexOrder) Len() int { return len(o.indices) }

func (o indexOrder) Less(first, second int) bool { return o.indices[first] < o.indices[second] }

func (o indexOrder) Swap(first, second int) {
	o.indices[first], o.indices[second] = o.indices[second], o.indices[first]
	o.weights[first], o.weights[second] = o.weights[second], o.weights[first]
}
