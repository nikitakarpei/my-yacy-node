package documentrelevance

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const leastRarityWhenNoQueryWordWasCounted = 1.0

type queryRarity struct {
	rarityPerQueryWord              map[yacymodel.Hash]float64
	rarityOfAQueryWordNobodyCounted float64
	sumAcrossTheQueryWords          float64
}

func queryRarityOf(
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
	queryWords []yacymodel.Hash,
) queryRarity {
	rarityPerQueryWord := rarityPerQueryWordFrom(documentsHeldPerQueryWord)
	rarityOfTheQuery := queryRarity{
		rarityPerQueryWord:              rarityPerQueryWord,
		rarityOfAQueryWordNobodyCounted: leastRarityCountedForAQueryWord(rarityPerQueryWord),
	}
	for _, word := range queryWords {
		rarityOfTheQuery.sumAcrossTheQueryWords += rarityOfTheQuery.rarityOfTheQueryWord(word)
	}

	return rarityOfTheQuery
}

func rarityPerQueryWordFrom(
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

func leastRarityCountedForAQueryWord(
	rarityPerQueryWord map[yacymodel.Hash]float64,
) float64 {
	leastRarityCounted := leastRarityWhenNoQueryWordWasCounted
	for _, rarityOfACountedQueryWord := range rarityPerQueryWord {
		leastRarityCounted = min(leastRarityCounted, rarityOfACountedQueryWord)
	}

	return leastRarityCounted
}

func (rarityOfTheQuery queryRarity) shareHeldBy(words []yacymodel.Hash) float64 {
	shareHeldByTheWords := 0.0
	for _, word := range words {
		shareHeldByTheWords += rarityOfTheQuery.shareHeldByTheQueryWord(word)
	}

	return shareHeldByTheWords
}

func (rarityOfTheQuery queryRarity) shareHeldByTheQueryWord(word yacymodel.Hash) float64 {
	return rarityOfTheQuery.rarityOfTheQueryWord(word) / rarityOfTheQuery.sumAcrossTheQueryWords
}

func (rarityOfTheQuery queryRarity) rarityOfTheQueryWord(word yacymodel.Hash) float64 {
	rarityOfTheQueryWord, counted := rarityOfTheQuery.rarityPerQueryWord[word]
	if !counted {
		return rarityOfTheQuery.rarityOfAQueryWordNobodyCounted
	}

	return rarityOfTheQueryWord
}
