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
	return itemsOrderedByRelevance(
		answers.ItemOfEachAnsweredDocument(), ordering.RelevancePerDocumentOf(answers),
	)
}

func (ordering Ordering) RelevancePerDocumentOf(
	answers peeranswers.AnsweredQuery,
) map[yacymodel.URLHash]float64 {
	items := answers.ItemOfEachAnsweredDocument()
	placeScorePerDocument := placeScorePerDocumentOf(answers.ItemsInTheOrderOfEachAnswer)
	rarityPerQueryWord := rarityPerQueryWordOf(answers.DocumentsHeldPerQueryWord)
	averageDocumentLength := averageDocumentLengthOf(items)

	relevancePerDocument := make(map[yacymodel.URLHash]float64, len(items))
	for _, item := range items {
		relevancePerDocument[item.Metadata.Hash] = ordering.scoreWeights.relevanceOf(
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
