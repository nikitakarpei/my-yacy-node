package relevance

import (
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	saturationOfTheHitsOfAWord           = 1.2
	weightOfTheDocumentLength            = 0.75
	lengthRatioOfADocumentNoPeerMeasured = 1.0
)

func textScoreOf(
	item peeranswers.AnsweredItem,
	rarity queryWordRarity,
	averageDocumentLength float64,
) float64 {
	amountOfTextWords := amountOfTextWordsOf(item)

	textScore := 0.0
	for _, word := range matchedWordsAlwaysInTheSameOrder(item) {
		textScore += rarity.rarityOfTheQueryWord(word) * saturatedHitsOf(
			item.MatchedWords[word].Hits, amountOfTextWords, averageDocumentLength,
		)
	}

	return textScore
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

func amountOfTextWordsOf(item peeranswers.AnsweredItem) int {
	amountOfTextWords := 0
	for _, count := range item.MatchedWords {
		amountOfTextWords = max(amountOfTextWords, count.TextWords)
	}

	return amountOfTextWords
}

func matchedWordsAlwaysInTheSameOrder(item peeranswers.AnsweredItem) []yacymodel.Hash {
	matchedWords := slices.Collect(maps.Keys(item.MatchedWords))
	slices.SortFunc(matchedWords, func(one, other yacymodel.Hash) int {
		return strings.Compare(one.String(), other.String())
	})

	return matchedWords
}

func saturatedHitsOf(hits int, amountOfTextWords int, averageDocumentLength float64) float64 {
	countedHits := float64(hits)
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
