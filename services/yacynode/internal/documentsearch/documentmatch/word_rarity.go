package documentmatch

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordRarity struct {
	words              []yacymodel.Hash
	rarityPerWord      map[yacymodel.Hash]float64
	rarestWordPosition int
	everyWordIsHeld    bool
}

func wordRarityOf(
	words []yacymodel.Hash,
	amountOfPostingsPerWord map[yacymodel.Hash]int,
) wordRarity {
	if len(words) == 0 {
		return wordRarity{}
	}
	largestAmountOfPostings := 0
	for _, word := range words {
		largestAmountOfPostings = max(largestAmountOfPostings, amountOfPostingsPerWord[word])
	}

	rarityPerWord := make(map[yacymodel.Hash]float64, len(words))
	rarestWordPosition := 0
	for position, word := range words {
		amountOfPostings := amountOfPostingsPerWord[word]
		if amountOfPostings <= 0 {
			return wordRarity{}
		}
		rarityPerWord[word] = math.Log1p(
			float64(largestAmountOfPostings) / float64(amountOfPostings),
		)
		if amountOfPostings < amountOfPostingsPerWord[words[rarestWordPosition]] {
			rarestWordPosition = position
		}
	}

	return wordRarity{
		words:              words,
		rarityPerWord:      rarityPerWord,
		rarestWordPosition: rarestWordPosition,
		everyWordIsHeld:    true,
	}
}

func (r wordRarity) rarestWord() yacymodel.Hash {
	return r.words[r.rarestWordPosition]
}

func (r wordRarity) rarityOf(word yacymodel.Hash) float64 {
	return r.rarityPerWord[word]
}
