package yacymodel

import (
	"maps"
	"slices"
	"strings"
)

type URLHashes map[URLHash]struct{}

func (hashes URLHashes) Contains(hash URLHash) bool {
	_, contained := hashes[hash]

	return contained
}

func (hashes URLHashes) Add(hash URLHash) {
	hashes[hash] = struct{}{}
}

func (hashes URLHashes) AddEach(addedHashes []URLHash) {
	for _, hash := range addedHashes {
		hashes.Add(hash)
	}
}

func (hashes URLHashes) InHashOrder() []URLHash {
	return slices.SortedFunc(maps.Keys(hashes), func(first, second URLHash) int {
		return strings.Compare(first.String(), second.String())
	})
}
