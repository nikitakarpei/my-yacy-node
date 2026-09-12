// Package hostdiscount orders the answered items by a relevance that falls by
// half for each item of the same host it already placed above. One host thus
// holds the whole first page only while its further items stay the most
// relevant ones. Items of equal discounted relevance keep the order of the
// ordering it wraps.
package hostdiscount

import (
	"math"
	"net/url"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const discountOfARepeatedHost = 0.5

type ItemsOrdering interface {
	OrderedItemsOf(answers peeranswers.AnsweredQuery) []peeranswers.AnsweredItem
	RelevancePerDocumentOf(answers peeranswers.AnsweredQuery) map[yacymodel.URLHash]float64
}

type Ordering struct {
	orderingByRelevance ItemsOrdering
}

func New(orderingByRelevance ItemsOrdering) Ordering {
	return Ordering{orderingByRelevance: orderingByRelevance}
}

func (o Ordering) OrderedItemsOf(
	answers peeranswers.AnsweredQuery,
) []peeranswers.AnsweredItem {
	return itemsInFallingOrderOfDiscountedRelevance(
		o.orderingByRelevance.OrderedItemsOf(answers),
		o.orderingByRelevance.RelevancePerDocumentOf(answers),
	)
}

func itemsInFallingOrderOfDiscountedRelevance(
	itemsInFallingOrderOfRelevance []peeranswers.AnsweredItem,
	relevancePerDocument map[yacymodel.URLHash]float64,
) []peeranswers.AnsweredItem {
	unplacedItems := itemsWithTheirHosts(itemsInFallingOrderOfRelevance)
	amountOfPlacedItemsPerHost := map[string]int{}
	placedItems := make([]peeranswers.AnsweredItem, 0, len(unplacedItems))
	for len(unplacedItems) > 0 {
		place := placeOfTheMostRelevantItemAfterTheDiscountAmong(
			unplacedItems, relevancePerDocument, amountOfPlacedItemsPerHost,
		)
		placedItems = append(placedItems, unplacedItems[place].item)
		amountOfPlacedItemsPerHost[unplacedItems[place].host]++
		unplacedItems = slices.Delete(unplacedItems, place, place+1)
	}

	return placedItems
}

type itemWithItsHost struct {
	item peeranswers.AnsweredItem
	host string
}

func itemsWithTheirHosts(items []peeranswers.AnsweredItem) []itemWithItsHost {
	itemsWithTheirHosts := make([]itemWithItsHost, 0, len(items))
	for _, item := range items {
		itemsWithTheirHosts = append(itemsWithTheirHosts, itemWithItsHost{
			item: item,
			host: hostOf(item.Metadata.Address),
		})
	}

	return itemsWithTheirHosts
}

func placeOfTheMostRelevantItemAfterTheDiscountAmong(
	items []itemWithItsHost,
	relevancePerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedItemsPerHost map[string]int,
) int {
	placeOfTheMostRelevantItem := 0
	highestDiscountedRelevance := math.Inf(-1)
	for place, item := range items {
		discountedRelevance := relevanceDiscountedPerHostOf(
			item, relevancePerDocument, amountOfPlacedItemsPerHost,
		)
		if discountedRelevance > highestDiscountedRelevance {
			highestDiscountedRelevance = discountedRelevance
			placeOfTheMostRelevantItem = place
		}
	}

	return placeOfTheMostRelevantItem
}

func relevanceDiscountedPerHostOf(
	item itemWithItsHost,
	relevancePerDocument map[yacymodel.URLHash]float64,
	amountOfPlacedItemsPerHost map[string]int,
) float64 {
	return relevancePerDocument[item.item.Metadata.Hash] * math.Pow(
		discountOfARepeatedHost,
		float64(amountOfPlacedItemsPerHost[item.host]),
	)
}

func hostOf(address string) string {
	readAddress, err := url.Parse(address)
	if err != nil || readAddress.Hostname() == "" {
		return address
	}

	return readAddress.Hostname()
}
