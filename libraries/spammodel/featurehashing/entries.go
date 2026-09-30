// Package featurehashing turns the features of a page into hashed entries the
// way the HashingVectorizer of scikit-learn does.
package featurehashing

import (
	"cmp"
	"math"
	"slices"
)

const Dimensions = 1 << 20

type Entry struct {
	Index int32
	Value float64
}

func EntriesFrom(features []string) []Entry {
	return normalizedEntriesFrom(countOfIndexFrom(features))
}

func countOfIndexFrom(features []string) map[int32]float64 {
	countOfIndex := make(map[int32]float64, len(features))
	for _, feature := range features {
		countOfIndex[indexOf(feature)]++
	}
	return countOfIndex
}

func indexOf(feature string) int32 {
	hash := int64(murmur3Of([]byte(feature)))
	return int32(max(hash, -hash) % Dimensions)
}

func normalizedEntriesFrom(countOfIndex map[int32]float64) []Entry {
	squaredLength := 0.0
	for _, count := range countOfIndex {
		squaredLength += count * count
	}
	length := math.Sqrt(squaredLength)
	entries := make([]Entry, 0, len(countOfIndex))
	for index, count := range countOfIndex {
		entries = append(entries, Entry{Index: index, Value: count / length})
	}
	slices.SortFunc(entries, func(first, second Entry) int {
		return cmp.Compare(first.Index, second.Index)
	})
	return entries
}
