package searchrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

type RelevanceBound struct {
	rarityOfRarestTerm             float64
	largestRelevanceFromOtherTerms Relevance
}

func RelevanceBoundOf(
	terms SearchTerms,
	largestImpactPerOtherTerm map[yacymodel.Hash]rwipostingimpactorder.Impact,
) RelevanceBound {
	largestRelevanceFromOtherTerms := Relevance(0)
	for _, term := range terms.OtherTerms() {
		largestRelevanceFromOtherTerms += Relevance(
			terms.rarityOf(term) * float64(largestImpactPerOtherTerm[term]),
		)
	}

	return RelevanceBound{
		rarityOfRarestTerm:             terms.rarityOf(terms.RarestTerm()),
		largestRelevanceFromOtherTerms: largestRelevanceFromOtherTerms,
	}
}

func (b RelevanceBound) LargestRelevanceReachableFrom(
	impactOfNextUnreadPosting rwipostingimpactorder.Impact,
) Relevance {
	return Relevance(b.rarityOfRarestTerm*float64(impactOfNextUnreadPosting)) +
		b.largestRelevanceFromOtherTerms
}
