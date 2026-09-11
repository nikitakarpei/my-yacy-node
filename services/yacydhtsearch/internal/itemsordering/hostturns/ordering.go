// Package hostturns orders the answered items so that the hosts take turns, and
// one host cannot hold the whole first page: the second item of a host comes
// after the first item of every other host, and so on. Within one turn the items
// keep the order of the ordering it wraps.
package hostturns

import (
	"net/url"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
)

type ItemsOrdering interface {
	OrderedItemsOf(answers peeranswers.AnsweredQuery) []peeranswers.AnsweredItem
}

type Ordering struct {
	orderingWithinATurn ItemsOrdering
}

func New(orderingWithinATurn ItemsOrdering) Ordering {
	return Ordering{orderingWithinATurn: orderingWithinATurn}
}

func (o Ordering) OrderedItemsOf(
	answers peeranswers.AnsweredQuery,
) []peeranswers.AnsweredItem {
	return itemsOfEveryTurnFrom(
		itemsOfEachTurnOf(o.orderingWithinATurn.OrderedItemsOf(answers)),
	)
}

func itemsOfEachTurnOf(items []peeranswers.AnsweredItem) [][]peeranswers.AnsweredItem {
	var itemsOfEachTurn [][]peeranswers.AnsweredItem
	amountOfItemsPerHost := map[string]int{}
	for _, item := range items {
		host := hostOf(item.Metadata.Address)
		turn := amountOfItemsPerHost[host]
		amountOfItemsPerHost[host] = turn + 1
		for turn >= len(itemsOfEachTurn) {
			itemsOfEachTurn = append(itemsOfEachTurn, nil)
		}
		itemsOfEachTurn[turn] = append(itemsOfEachTurn[turn], item)
	}

	return itemsOfEachTurn
}

func hostOf(address string) string {
	readAddress, err := url.Parse(address)
	if err != nil || readAddress.Hostname() == "" {
		return address
	}

	return readAddress.Hostname()
}

func itemsOfEveryTurnFrom(
	itemsOfEachTurn [][]peeranswers.AnsweredItem,
) []peeranswers.AnsweredItem {
	var itemsOfEveryTurn []peeranswers.AnsweredItem
	for _, itemsOfOneTurn := range itemsOfEachTurn {
		itemsOfEveryTurn = append(itemsOfEveryTurn, itemsOfOneTurn...)
	}

	return itemsOfEveryTurn
}
