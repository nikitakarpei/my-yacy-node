// Package relevance orders the answered items by how well each document
// answers the query. Four scores add up to its relevance: how high the
// answering peers put it, where each answering peer adds most of a place
// score and its place adds the rest, how many query words its title holds,
// how often they appear in its text, and how many of them the host of its
// address holds. A first place from one peer weighs one, a query word in the
// title weighs one half, a missing title weighs minus one, a query word in
// the host weighs one, and one saturated hit weighs the rarity of its word.
// An address that does not parse holds no word. The text score follows BM25:
// a word weighs more the fewer documents the peers hold for it and the more
// often the page of the document or a peer counted it, each further hit
// adding less, and weighs less the longer the document is. A word no peer
// counted documents for weighs as much as the most common counted word, a
// word no page and no peer counted hits of weighs nothing, and a document of
// unmeasured length is of average length.
// Documents of equal relevance keep the order the peers put them in.
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
		weightOfTheAddressScore*addressScoreOf(item)
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
