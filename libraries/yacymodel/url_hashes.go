package yacymodel

import (
	"iter"
	"slices"
)

type URLHashes struct {
	inHashOrder []URLHash
}

func URLHashesOf(hashes ...URLHash) URLHashes {
	var builder URLHashesBuilder
	builder.Grow(len(hashes))
	for _, hash := range hashes {
		builder.Add(hash)
	}

	return builder.URLHashes()
}

func (hashes URLHashes) Contains(hash URLHash) bool {
	_, found := slices.BinarySearchFunc(hashes.inHashOrder, hash, URLHash.CompareInAlphabetOrder)

	return found
}

func (hashes URLHashes) Amount() int {
	return len(hashes.inHashOrder)
}

func (hashes URLHashes) All() iter.Seq[URLHash] {
	return slices.Values(hashes.inHashOrder)
}

func (hashes URLHashes) InHashOrder() []URLHash {
	return slices.Clone(hashes.inHashOrder)
}

func (hashes URLHashes) UnionWith(others ...URLHashes) URLHashes {
	var builder URLHashesBuilder
	builder.Grow(hashes.Amount() + amountInEachOf(others))
	builder.AddEach(hashes)
	for _, other := range others {
		builder.AddEach(other)
	}

	return builder.URLHashes()
}

func amountInEachOf(sets []URLHashes) int {
	amount := 0
	for _, set := range sets {
		amount += set.Amount()
	}

	return amount
}

func (hashes URLHashes) Without(excluded URLHashes) URLHashes {
	var builder URLHashesBuilder
	for _, hash := range hashes.inHashOrder {
		if !excluded.Contains(hash) {
			builder.Add(hash)
		}
	}

	return builder.URLHashes()
}

func (hashes URLHashes) between(lowest, highest URLHash) URLHashes {
	start, _ := slices.BinarySearchFunc(hashes.inHashOrder, lowest, URLHash.CompareInAlphabetOrder)
	end, found := slices.BinarySearchFunc(
		hashes.inHashOrder,
		highest,
		URLHash.CompareInAlphabetOrder,
	)
	if found {
		end++
	}

	return URLHashes{inHashOrder: hashes.inHashOrder[start:end:end]}
}
