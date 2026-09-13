// Package indexabstract owns the index abstracts of one search request: which
// terms they cover, and the documents each one names, the most relevant first
// and up to a cap, read from the index of this node. A peer reads them to plan
// which peers to ask next, so they carry document hashes only.
package indexabstract

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type IndexAbstracts map[yacymodel.Hash][]yacymodel.URLHash

func termsInIndex(
	queryTerms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) []yacymodel.Hash {
	return slices.DeleteFunc(
		slices.Clone(queryTerms),
		func(term yacymodel.Hash) bool { return amountOfPostingsPerTerm[term] <= 0 },
	)
}
