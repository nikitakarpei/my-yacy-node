// Package indexabstract owns the index abstracts of one search request: the
// terms they cover, and the documents each one names. A peer reads them to
// plan which peers to ask next, so they carry document hashes only, never
// metadata.
package indexabstract

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type IndexAbstracts map[yacymodel.Hash][]yacymodel.URLHash

func IndexAbstractsOf(
	terms []yacymodel.Hash,
	documentsPerTerm map[yacymodel.Hash][]yacymodel.URLHash,
) IndexAbstracts {
	abstracts := make(IndexAbstracts, len(terms))
	for _, term := range terms {
		abstracts[term] = documentsPerTerm[term]
	}

	return abstracts
}

func termsInIndex(
	queryTerms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) []yacymodel.Hash {
	return slices.DeleteFunc(
		slices.Clone(queryTerms),
		func(term yacymodel.Hash) bool { return amountOfPostingsPerTerm[term] <= 0 },
	)
}
