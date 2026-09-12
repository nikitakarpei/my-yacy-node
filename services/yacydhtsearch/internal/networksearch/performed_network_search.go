package networksearch

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedNetworkSearch struct {
	AmountOfAskablePeers              int
	AmountOfItemsAcrossAnswers        int
	AmountOfItemsInRanking            int
	AmountOfRankedItemsOfTheOnePeer   int
	AmountOfRankedItemsCountedByAPeer int
	TimeSpent                         time.Duration
}

func performedNetworkSearchFrom(
	answers peeranswers.AnsweredQuery,
	rankedItems []peeranswers.AnsweredItem,
	amountOfAskablePeers int,
	timeSpent time.Duration,
) PerformedNetworkSearch {
	return PerformedNetworkSearch{
		AmountOfAskablePeers:       amountOfAskablePeers,
		AmountOfItemsAcrossAnswers: amountOfItemsAcrossAnswers(answers),
		AmountOfItemsInRanking:     len(rankedItems),
		AmountOfRankedItemsOfTheOnePeer: amountOfRankedItemsOfTheOnePeer(
			answers.ItemsInTheOrderOfEachPeerRanking,
			rankedItems,
		),
		AmountOfRankedItemsCountedByAPeer: amountOfItemsCountedByAPeer(rankedItems),
		TimeSpent:                         timeSpent,
	}
}

func amountOfItemsAcrossAnswers(answers peeranswers.AnsweredQuery) int {
	amountOfAnsweredItems := len(answers.ItemsInNoOrder)
	for _, itemsOfOnePeerRanking := range answers.ItemsInTheOrderOfEachPeerRanking {
		amountOfAnsweredItems += len(itemsOfOnePeerRanking)
	}

	return amountOfAnsweredItems
}

func amountOfRankedItemsOfTheOnePeer(
	itemsInTheOrderOfEachPeerRanking [][]peeranswers.AnsweredItem,
	rankedItems []peeranswers.AnsweredItem,
) int {
	rankedDocuments := make(map[yacymodel.URLHash]struct{}, len(rankedItems))
	for _, item := range rankedItems {
		rankedDocuments[item.Metadata.Hash] = struct{}{}
	}

	var mostRankedItemsOfOnePeer int
	for _, items := range itemsInTheOrderOfEachPeerRanking {
		mostRankedItemsOfOnePeer = max(
			mostRankedItemsOfOnePeer,
			amountOfRankedItemsAmong(items, rankedDocuments),
		)
	}

	return mostRankedItemsOfOnePeer
}

func amountOfRankedItemsAmong(
	items []peeranswers.AnsweredItem,
	rankedDocuments map[yacymodel.URLHash]struct{},
) int {
	countedDocuments := make(map[yacymodel.URLHash]struct{}, len(items))
	for _, item := range items {
		if _, ranked := rankedDocuments[item.Metadata.Hash]; !ranked {
			continue
		}
		countedDocuments[item.Metadata.Hash] = struct{}{}
	}

	return len(countedDocuments)
}

func amountOfItemsCountedByAPeer(items []peeranswers.AnsweredItem) int {
	amount := 0
	for _, item := range items {
		if !item.CountedByAPeer() {
			continue
		}
		amount++
	}

	return amount
}
