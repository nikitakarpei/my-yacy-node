package yacymodel

import "slices"

type URLHashesBuilder struct {
	added []URLHash
}

func (builder *URLHashesBuilder) Grow(amount int) {
	builder.added = slices.Grow(builder.added, amount)
}

func (builder *URLHashesBuilder) Add(hash URLHash) {
	builder.added = append(builder.added, hash)
}

func (builder *URLHashesBuilder) AddEach(hashes URLHashes) {
	builder.added = append(builder.added, hashes.inHashOrder...)
}

func (builder *URLHashesBuilder) URLHashes() URLHashes {
	inHashOrder := builder.added
	builder.added = nil
	slices.SortFunc(inHashOrder, URLHash.CompareInAlphabetOrder)
	inHashOrder = slices.Compact(inHashOrder)
	if len(inHashOrder) == 0 {
		return URLHashes{}
	}
	if unusedCapacity := cap(inHashOrder) - len(inHashOrder); unusedCapacity > len(inHashOrder)/4 {
		inHashOrder = slices.Clone(inHashOrder)
	}

	return URLHashes{inHashOrder: slices.Clip(inHashOrder)}
}
