package relevance

import (
	"math"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	rarityOfAQueryWordWhenNoWordWasCounted = 1.0
	hitsOfAWordNoPeerCounted               = 1
	saturationOfTheHitsOfAWord             = 1.2
	weightOfTheDocumentLength              = 0.75
	lengthRatioOfADocumentNoPeerMeasured   = 1.0
)

func textScoreOf(
	item peeranswers.AnsweredItem,
	rarityPerQueryWord map[yacymodel.Hash]float64,
	averageDocumentLength float64,
) float64 {
	amountOfTextWords := amountOfTextWordsOf(item)
	rarityOfAnUncountedQueryWord := rarityOfAnUncountedQueryWordFrom(rarityPerQueryWord)

	textScore := 0.0
	for word, count := range item.MatchedWords {
		rarity, counted := rarityPerQueryWord[word]
		if !counted {
			rarity = rarityOfAnUncountedQueryWord
		}
		textScore += rarity * saturatedHitsOf(count.Hits, amountOfTextWords, averageDocumentLength)
	}

	return textScore
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

func averageDocumentLengthOf(items []peeranswers.AnsweredItem) float64 {
	sumOfTextWordsAcrossDocuments, amountOfMeasuredDocuments := 0, 0
	for _, item := range items {
		amountOfTextWords := amountOfTextWordsOf(item)
		if amountOfTextWords <= 0 {
			continue
		}
		sumOfTextWordsAcrossDocuments += amountOfTextWords
		amountOfMeasuredDocuments++
	}
	if amountOfMeasuredDocuments == 0 {
		return 0
	}

	return float64(sumOfTextWordsAcrossDocuments) / float64(amountOfMeasuredDocuments)
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

func amountOfTextWordsOf(item peeranswers.AnsweredItem) int {
	amountOfTextWords := 0
	for _, count := range item.MatchedWords {
		amountOfTextWords = max(amountOfTextWords, count.TextWords)
	}

	return amountOfTextWords
}

func saturatedHitsOf(hits int, amountOfTextWords int, averageDocumentLength float64) float64 {
	countedHits := float64(max(hits, hitsOfAWordNoPeerCounted))
	saturationForTheDocumentLength := saturationOfTheHitsOfAWord * (1 - weightOfTheDocumentLength +
		weightOfTheDocumentLength*documentLengthRatioOf(amountOfTextWords, averageDocumentLength))

	return countedHits * (saturationOfTheHitsOfAWord + 1) /
		(countedHits + saturationForTheDocumentLength)
}

func documentLengthRatioOf(amountOfTextWords int, averageDocumentLength float64) float64 {
	if amountOfTextWords <= 0 || averageDocumentLength <= 0 {
		return lengthRatioOfADocumentNoPeerMeasured
	}

	return float64(amountOfTextWords) / averageDocumentLength
}
