// Package indexabstract chooses the terms the index abstracts of one search
// request cover, and holds the abstracts built from them. One index abstract
// names a term and the documents this node holds for it. A peer reads index
// abstracts to plan which peers to ask next, so they carry document hashes
// only, never metadata.
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
