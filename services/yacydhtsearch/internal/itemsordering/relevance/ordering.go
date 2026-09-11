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

type Ordering struct {
	scoreWeights ScoreWeights
}

func New(scoreWeights ScoreWeights) Ordering {
	return Ordering{scoreWeights: scoreWeights}
}

func (ordering Ordering) OrderedItemsOf(
	answers peeranswers.AnsweredQuery,
) []peeranswers.AnsweredItem {
	items := answers.ItemOfEachAnsweredDocument()
	relevancePerDocument := relevancePerDocumentOf(
		items,
		answers.ItemsInTheOrderOfEachAnswer,
		answers.DocumentsHeldPerQueryWord,
		ordering.scoreWeights,
	)

	return itemsOrderedByRelevance(items, relevancePerDocument)
}

func relevancePerDocumentOf(
	items []peeranswers.AnsweredItem,
	itemsInTheOrderOfEachAnswer [][]peeranswers.AnsweredItem,
	documentsHeldPerQueryWord map[yacymodel.Hash]int,
	scoreWeights ScoreWeights,
) map[yacymodel.URLHash]float64 {
	placeScorePerDocument := placeScorePerDocumentOf(itemsInTheOrderOfEachAnswer)
	rarityPerQueryWord := rarityPerQueryWordOf(documentsHeldPerQueryWord)
	averageDocumentLength := averageDocumentLengthOf(items)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(items))
	for _, item := range items {
		relevancePerDocument[item.Metadata.Hash] = scoreWeights.relevanceOf(
			item,
			placeScorePerDocument[item.Metadata.Hash],
			rarityPerQueryWord,
			averageDocumentLength,
		)
	}

	return relevancePerDocument
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
