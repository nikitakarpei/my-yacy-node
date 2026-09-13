package indexabstract

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type IndexAbstractOfTermWithMostPostings struct{}

func (IndexAbstractOfTermWithMostPostings) coveredTerms(
	queryTerms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) []yacymodel.Hash {
	terms := termsInIndex(queryTerms, amountOfPostingsPerTerm)
	if len(terms) == 0 {
		return nil
	}

	return []yacymodel.Hash{
		slices.MinFunc(terms, func(a, b yacymodel.Hash) int {
			return cmp.Or(
				cmp.Compare(amountOfPostingsPerTerm[b], amountOfPostingsPerTerm[a]),
				yacymodel.CompareInAlphabetOrder(a.String(), b.String()),
			)
		}),
	}
}
