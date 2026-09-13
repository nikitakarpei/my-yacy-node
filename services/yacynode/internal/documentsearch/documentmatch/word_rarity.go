package documentmatch

import (
	"math"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordRarity struct {
	rarestWord               yacymodel.Hash
	wordsBesideTheRarestWord []yacymodel.Hash
	rarityPerWord            map[yacymodel.Hash]float64
	everyWordIsHeld          bool
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
		if amountOfPostingsPerWord[word] <= 0 {
			return wordRarity{}
		}
		largestAmountOfPostings = max(largestAmountOfPostings, amountOfPostingsPerWord[word])
	}
	rarestWord := rarestWordOf(words, amountOfPostingsPerWord)

	return wordRarity{
		rarestWord:               rarestWord,
		wordsBesideTheRarestWord: wordsBesideTheRarestWordOf(words, rarestWord),
		rarityPerWord: rarityPerWordOf(
			words,
			amountOfPostingsPerWord,
			largestAmountOfPostings,
		),
		everyWordIsHeld: true,
	}
}

func rarestWordOf(
	words []yacymodel.Hash,
	amountOfPostingsPerWord map[yacymodel.Hash]int,
) yacymodel.Hash {
	rarestWord := words[0]
	for _, word := range words {
		if amountOfPostingsPerWord[word] < amountOfPostingsPerWord[rarestWord] {
			rarestWord = word
		}
	}

	return rarestWord
}

func wordsBesideTheRarestWordOf(
	words []yacymodel.Hash,
	rarestWord yacymodel.Hash,
) []yacymodel.Hash {
	rarestWordAt := slices.Index(words, rarestWord)

	return slices.Concat(words[:rarestWordAt], words[rarestWordAt+1:])
}

func rarityPerWordOf(
	words []yacymodel.Hash,
	amountOfPostingsPerWord map[yacymodel.Hash]int,
	largestAmountOfPostings int,
) map[yacymodel.Hash]float64 {
	rarityPerWord := make(map[yacymodel.Hash]float64, len(words))
	for _, word := range words {
		rarityPerWord[word] = math.Log1p(
			float64(largestAmountOfPostings) / float64(amountOfPostingsPerWord[word]),
		)
	}

	return rarityPerWord
}

func (r wordRarity) rarityOf(word yacymodel.Hash) float64 {
	return r.rarityPerWord[word]
}
