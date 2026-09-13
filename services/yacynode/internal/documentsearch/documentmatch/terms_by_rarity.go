package documentmatch

import (
	"math"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type termsByRarity struct {
	rarestTerm    yacymodel.Hash
	otherTerms    []yacymodel.Hash
	rarityPerTerm map[yacymodel.Hash]float64
}

func termsByRarityOf(
	terms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) termsByRarity {
	largestAmountOfPostings := 0
	for _, term := range terms {
		largestAmountOfPostings = max(largestAmountOfPostings, amountOfPostingsPerTerm[term])
	}
	rarestTerm := rarestTermOf(terms, amountOfPostingsPerTerm)

	return termsByRarity{
		rarestTerm: rarestTerm,
		otherTerms: otherTermsOf(terms, rarestTerm),
		rarityPerTerm: rarityPerTermOf(
			terms,
			amountOfPostingsPerTerm,
			largestAmountOfPostings,
		),
	}
}

func rarestTermOf(
	terms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
) yacymodel.Hash {
	rarestTerm := terms[0]
	for _, term := range terms {
		if amountOfPostingsPerTerm[term] < amountOfPostingsPerTerm[rarestTerm] {
			rarestTerm = term
		}
	}

	return rarestTerm
}

func otherTermsOf(
	terms []yacymodel.Hash,
	rarestTerm yacymodel.Hash,
) []yacymodel.Hash {
	rarestTermAt := slices.Index(terms, rarestTerm)

	return slices.Concat(terms[:rarestTermAt], terms[rarestTermAt+1:])
}

func rarityPerTermOf(
	terms []yacymodel.Hash,
	amountOfPostingsPerTerm map[yacymodel.Hash]int,
	largestAmountOfPostings int,
) map[yacymodel.Hash]float64 {
	rarityPerTerm := make(map[yacymodel.Hash]float64, len(terms))
	for _, term := range terms {
		rarityPerTerm[term] = math.Log1p(
			float64(largestAmountOfPostings) / float64(amountOfPostingsPerTerm[term]),
		)
	}

	return rarityPerTerm
}

func (t termsByRarity) rarityOf(term yacymodel.Hash) float64 {
	return t.rarityPerTerm[term]
}
