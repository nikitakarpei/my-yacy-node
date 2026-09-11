// Package relevance orders the answered items by how well each document
// answers the query. Its relevance adds up how high the peers placed it, the
// query words its title and the host of its address hold, a BM25 score of the
// query words in its text, and the query phrases its text holds. Documents of
// equal relevance keep the order the peers put them in.
package relevance

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	weightOfThePlaceScore   = 1.0
	weightOfTheTitleScore   = 0.5
	weightOfTheTextScore    = 1.0
	weightOfTheAddressScore = 1.0
	weightOfThePhraseScore  = 1.0
)

type Ordering struct{}

func (Ordering) OrderedItemsOf(
	answers peeranswers.AnsweredQuery,
) []peeranswers.AnsweredItem {
	items := answers.ItemOfEachAnsweredDocument()
	relevancePerDocument := relevancePerDocumentOf(
		items, answers.ItemsInTheOrderOfEachAnswer, answers.DocumentsHeldPerQueryWord,
	)

	return itemsOrderedByRelevance(items, relevancePerDocument)
}

func relevancePerDocumentOf(
	items []peeranswers.AnsweredItem,
	itemsInTheOrderOfEachAnswer [][]peeranswers.AnsweredItem,
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
) map[yacymodel.URLHash]float64 {
	placeScorePerDocument := placeScorePerDocumentOf(itemsInTheOrderOfEachAnswer)
	rarityPerQueryWord := rarityPerQueryWordOf(documentsHeldPerQueryWord)
	averageDocumentLength := averageDocumentLengthOf(items)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(items))
	for _, item := range items {
		relevancePerDocument[item.Metadata.Hash] = relevanceOf(
			item,
			placeScorePerDocument[item.Metadata.Hash],
			rarityPerQueryWord,
			averageDocumentLength,
		)
	}

	return relevancePerDocument
}

func relevanceOf(
	item peeranswers.AnsweredItem,
	placeScore float64,
	rarityPerQueryWord map[yacymodel.Hash]float64,
	averageDocumentLength float64,
) float64 {
	return weightOfThePlaceScore*placeScore +
		weightOfTheTitleScore*titleScoreOf(item) +
		weightOfTheTextScore*textScoreOf(item, rarityPerQueryWord, averageDocumentLength) +
		weightOfTheAddressScore*addressScoreOf(item) +
		weightOfThePhraseScore*phraseScoreOf(item)
}

func itemsOrderedByRelevance(
	items []peeranswers.AnsweredItem,
	relevancePerDocument map[yacymodel.URLHash]float64,
) []peeranswers.AnsweredItem {
	slices.SortStableFunc(items, func(one, other peeranswers.AnsweredItem) int {
		return cmp.Compare(
			relevancePerDocument[other.Metadata.Hash], relevancePerDocument[one.Metadata.Hash],
		)
	})

	return items
}
