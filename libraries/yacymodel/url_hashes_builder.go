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

func (builder *URLHashesBuilder) URLHashes() URLHashes {
	inHashOrder := builder.added
	builder.added = nil
	slices.SortFunc(inHashOrder, URLHash.CompareInAlphabetOrder)

	return URLHashes{inHashOrder: withoutSpareCapacity(slices.Compact(inHashOrder))}
}
