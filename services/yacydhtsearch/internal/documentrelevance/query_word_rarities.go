package documentrelevance

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const rarityWhenNoWordCounted = 1.0

type queryWordRarities struct {
	rarityPerWord         map[yacymodel.Hash]float64
	rarityOfUncountedWord float64
	sumOfRarities         float64
}

func queryWordRaritiesFrom(answers queryanswers.AnsweredQuery) queryWordRarities {
	rarityPerWord := rarityPerWordFrom(answers.DocumentsHeldPerQueryWord)
	rarities := queryWordRarities{
		rarityPerWord:         rarityPerWord,
		rarityOfUncountedWord: leastRarityAmong(rarityPerWord),
	}
	for _, word := range answers.QueryWords {
		rarities.sumOfRarities += rarities.rarityOf(word)
	}

	return rarities
}

func rarityPerWordFrom(
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
) map[yacymodel.Hash]float64 {
	mostDocumentsHeldForQueryWord := 0
	for _, documentsHeld := range documentsHeldPerQueryWord {
		mostDocumentsHeldForQueryWord = max(mostDocumentsHeldForQueryWord, documentsHeld)
	}

	rarityPerWord := make(map[yacymodel.Hash]float64, len(documentsHeldPerQueryWord))
	for word, documentsHeld := range documentsHeldPerQueryWord {
		if documentsHeld <= 0 {
			continue
		}
		rarityPerWord[word] = math.Log1p(
			float64(mostDocumentsHeldForQueryWord) / float64(documentsHeld),
		)
	}

	return rarityPerWord
}

func leastRarityAmong(rarityPerWord map[yacymodel.Hash]float64) float64 {
	if len(rarityPerWord) == 0 {
		return rarityWhenNoWordCounted
	}

	leastRarity := math.Inf(1)
	for _, rarity := range rarityPerWord {
		leastRarity = min(leastRarity, rarity)
	}

	return leastRarity
}

func (rarities queryWordRarities) rarityShareOfWords(words []yacymodel.Hash) float64 {
	rarityShare := 0.0
	for _, word := range words {
		rarityShare += rarities.rarityShareOfWord(word)
	}

	return rarityShare
}

func (rarities queryWordRarities) rarityShareOfWord(word yacymodel.Hash) float64 {
	return rarities.rarityOf(word) / rarities.sumOfRarities
}

func (rarities queryWordRarities) rarityOf(word yacymodel.Hash) float64 {
	countedRarity, rarityCounted := rarities.rarityPerWord[word]
	if !rarityCounted {
		return rarities.rarityOfUncountedWord
	}

	return countedRarity
}
