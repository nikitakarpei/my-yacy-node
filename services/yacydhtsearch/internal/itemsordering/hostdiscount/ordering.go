// Package hostdiscount orders the answered items by a relevance that falls by
// half for each item of the same host it already placed above. One host thus
// holds the whole first page only while its further items stay the most
// relevant ones. Items of equal discounted relevance keep the order of falling
// relevance, in which documents of equal relevance keep the order of the peers.
package hostdiscount

import (
	"cmp"
	"math"
	"net/url"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const shareOfRelevanceKeptPerPlacedItemOfTheSameHost = 0.5

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
	relevancePerDocument := ordering.documentRelevance.RelevancePerDocumentOf(answers)

	return itemsInFallingOrderOfDiscountedRelevance(
		itemsInFallingOrderOfRelevance(
			answers.ItemOfEachAnsweredDocument(), relevancePerDocument,
		),
		relevancePerDocument,
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

func itemsInFallingOrderOfDiscountedRelevance(
	itemsOfFallingRelevance []queryanswers.AnsweredItem,
	relevancePerDocument map[yacymodel.URLHash]float64,
) []queryanswers.AnsweredItem {
	unplacedHostedItems := hostedItemsOf(itemsOfFallingRelevance)
	amountOfPlacedItemsPerHost := map[string]int{}
	placedItems := make([]queryanswers.AnsweredItem, 0, len(unplacedHostedItems))
	for len(unplacedHostedItems) > 0 {
		position := positionOfTheHighestDiscountedRelevanceAmong(
			unplacedHostedItems, relevancePerDocument, amountOfPlacedItemsPerHost,
		)
		placedItems = append(placedItems, unplacedHostedItems[position].item)
		amountOfPlacedItemsPerHost[unplacedHostedItems[position].host]++
		unplacedHostedItems = slices.Delete(unplacedHostedItems, position, position+1)
	}

	return placedItems
}

type hostedItem struct {
	item queryanswers.AnsweredItem
	host string
}

func hostedItemsOf(items []queryanswers.AnsweredItem) []hostedItem {
	hostedItems := make([]hostedItem, 0, len(items))
	for _, item := range items {
		hostedItems = append(hostedItems, hostedItem{
			item: item,
			host: hostOf(item.Metadata.Address),
		})
	}

	return hostedItems
}

func positionOfTheHighestDiscountedRelevanceAmong(
	hostedItems []hostedItem,
	relevancePerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedItemsPerHost map[string]int,
) int {
	positionOfTheHighestDiscountedRelevance := 0
	highestDiscountedRelevance := math.Inf(-1)
	for position, hostedItem := range hostedItems {
		discountedRelevance := discountedRelevanceOf(
			hostedItem, relevancePerDocument, amountOfPlacedItemsPerHost,
		)
		if discountedRelevance > highestDiscountedRelevance {
			highestDiscountedRelevance = discountedRelevance
			positionOfTheHighestDiscountedRelevance = position
		}
	}

	return positionOfTheHighestDiscountedRelevance
}

func discountedRelevanceOf(
	hostedItem hostedItem,
	relevancePerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedItemsPerHost map[string]int,
) float64 {
	return relevancePerDocument[hostedItem.item.Metadata.Hash] * math.Pow(
		shareOfRelevanceKeptPerPlacedItemOfTheSameHost,
		float64(amountOfPlacedItemsPerHost[hostedItem.host]),
	)
}

func hostOf(address string) string {
	parsedAddress, err := url.Parse(address)
	if err != nil || parsedAddress.Hostname() == "" {
		return address
	}

	return parsedAddress.Hostname()
}
