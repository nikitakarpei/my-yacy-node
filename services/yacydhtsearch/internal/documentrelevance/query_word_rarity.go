package documentrelevance

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const rarityOfAQueryWordWhenNoWordWasCounted = 1.0

type queryWordRarity struct {
	rarityPerQueryWord            map[yacymodel.Hash]float64
	rarityOfAnUncountedQueryWord  float64
	sumOfTheRarityOfTheQueryWords float64
}

func queryWordRarityOf(
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
	queryWordsOfTheAnswers []yacymodel.Hash,
) queryWordRarity {
	rarityPerQueryWord := rarityPerQueryWordOf(documentsHeldPerQueryWord)
	rarity := queryWordRarity{
		rarityPerQueryWord:           rarityPerQueryWord,
		rarityOfAnUncountedQueryWord: rarityOfAnUncountedQueryWordFrom(rarityPerQueryWord),
	}
	rarity.sumOfTheRarityOfTheQueryWords = rarity.sumOfTheRarityOf(queryWordsOfTheAnswers)

	return rarity
}

func rarityPerQueryWordOf(
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
) map[yacymodel.Hash]float64 {
	mostDocumentsHeldForAQueryWord := 0
	for _, documentsHeld := range documentsHeldPerQueryWord {
		mostDocumentsHeldForAQueryWord = max(mostDocumentsHeldForAQueryWord, documentsHeld)
	}

	rarityPerQueryWord := make(map[yacymodel.Hash]float64, len(documentsHeldPerQueryWord))
	for word, documentsHeld := range documentsHeldPerQueryWord {
		if documentsHeld <= 0 {
			continue
		}
		rarityPerQueryWord[word] = math.Log1p(
			float64(mostDocumentsHeldForAQueryWord) / float64(documentsHeld),
		)
	}

	return rarityPerQueryWord
}

func rarityOfAnUncountedQueryWordFrom(
	rarityPerQueryWord map[yacymodel.Hash]float64,
) float64 {
	rarityOfAnUncountedQueryWord := rarityOfAQueryWordWhenNoWordWasCounted
	for _, rarityOfACountedQueryWord := range rarityPerQueryWord {
		rarityOfAnUncountedQueryWord = min(rarityOfAnUncountedQueryWord, rarityOfACountedQueryWord)
	}

	return rarityOfAnUncountedQueryWord
}

func (rarity queryWordRarity) sumOfTheRarityOf(queryWords []yacymodel.Hash) float64 {
	sumOfTheRarityOfTheQueryWords := 0.0
	for _, word := range queryWords {
		sumOfTheRarityOfTheQueryWords += rarity.rarityOfTheQueryWord(word)
	}

	return sumOfTheRarityOfTheQueryWords
}

func (rarity queryWordRarity) rarityOfTheQueryWord(word yacymodel.Hash) float64 {
	rarityOfTheQueryWord, counted := rarity.rarityPerQueryWord[word]
	if !counted {
		return rarity.rarityOfAnUncountedQueryWord
	}

	return rarityOfTheQueryWord
}
