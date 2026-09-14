package indexabstract

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type RequestedIndexAbstract interface {
	coveredTerms(
		queryTerms []yacymodel.Hash,
		amountOfPostingsPerTerm map[yacymodel.Hash]int,
	) []yacymodel.Hash
}

type RequestedIndexAbstracts []RequestedIndexAbstract

func termsCoveredBy(
	requested RequestedIndexAbstracts,
	queryTerms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) []yacymodel.Hash {
	var terms []yacymodel.Hash
	for _, requestedAbstract := range requested {
		for _, term := range requestedAbstract.coveredTerms(
			queryTerms,
			amountOfPostingsPerTerm,
		) {
			if !slices.Contains(terms, term) {
				terms = append(terms, term)
			}
		}
	}

	return terms
}
