// Package relevance orders the answered items by how well each document
// answers the query, the most relevant document first. Documents of equal
// relevance keep the order the peers put them in.
package relevance

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type DocumentRelevance interface {
	RelevancePerDocumentOf(answers queryanswers.AnsweredQuery) map[yacymodel.URLHash]float64
}

type Ordering struct {
	documentRelevance DocumentRelevance
}

func New(documentRelevance DocumentRelevance) Ordering {
	return Ordering{documentRelevance: documentRelevance}
}

func (ordering Ordering) OrderedItemsOf(
	answers queryanswers.AnsweredQuery,
) []queryanswers.AnsweredItem {
	return itemsInFallingOrderOfRelevance(
		answers.ItemOfEachAnsweredDocument(),
		ordering.documentRelevance.RelevancePerDocumentOf(answers),
	)
}

func itemsInFallingOrderOfRelevance(
	items []queryanswers.AnsweredItem,
	relevancePerDocument map[yacymodel.URLHash]float64,
) []queryanswers.AnsweredItem {
	slices.SortStableFunc(items, func(one, other queryanswers.AnsweredItem) int {
		return cmp.Compare(
			relevancePerDocument[other.Metadata.Hash], relevancePerDocument[one.Metadata.Hash],
		)
	})

	return items
}
