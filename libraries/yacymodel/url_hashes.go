package yacymodel

import (
	"iter"
	"slices"
)

type URLHashes struct {
	inHashOrder []URLHash
}

func URLHashesFrom(hashes ...URLHash) URLHashes {
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

func UnionOf(sets ...URLHashes) URLHashes {
	unmerged := slices.DeleteFunc(slices.Clone(sets), func(set URLHashes) bool {
		return set.Amount() == 0
	})
	if len(unmerged) == 0 {
		return URLHashes{}
	}
	for len(unmerged) > 1 {
		merged := make([]URLHashes, 0, (len(unmerged)+1)/2)
		for first := 0; first+1 < len(unmerged); first += 2 {
			merged = append(merged, unionOfTwo(unmerged[first], unmerged[first+1]))
		}
		if len(unmerged)%2 == 1 {
			merged = append(merged, unmerged[len(unmerged)-1])
		}
		unmerged = merged
	}

	return unmerged[0]
}

func unionOfTwo(first, second URLHashes) URLHashes {
	union := make([]URLHash, 0, first.Amount()+second.Amount())
	firstLeft, secondLeft := first.inHashOrder, second.inHashOrder
	for len(firstLeft) > 0 && len(secondLeft) > 0 {
		order := firstLeft[0].CompareInAlphabetOrder(secondLeft[0])
		if order <= 0 {
			union = append(union, firstLeft[0])
			firstLeft = firstLeft[1:]
		}
		if order >= 0 {
			if order > 0 {
				union = append(union, secondLeft[0])
			}
			secondLeft = secondLeft[1:]
		}
	}
	union = append(union, firstLeft...)

	return urlHashesHolding(append(union, secondLeft...))
}

func urlHashesHolding(inHashOrder []URLHash) URLHashes {
	if len(inHashOrder) == 0 {
		return URLHashes{}
	}
	if unusedCapacity := cap(inHashOrder) - len(inHashOrder); unusedCapacity > len(inHashOrder)/4 {
		inHashOrder = slices.Clone(inHashOrder)
	}

	return URLHashes{inHashOrder: slices.Clip(inHashOrder)}
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
