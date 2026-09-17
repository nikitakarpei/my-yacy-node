package networksearch

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedNetworkSearch struct {
	AmountOfAskablePeers                   int
	AmountOfItemsAcrossAnswers             int
	AmountOfItemsInRanking                 int
	AmountOfRankedItemsOfTheMostRankedPeer int
	AmountOfRankedItemsCountedByAPeer      int
	TimeSpent                              time.Duration
}

func performedNetworkSearchFrom(
	answers queryanswers.AnsweredQuery,
	rankedItems []queryanswers.AnsweredItem,
	amountOfAskablePeers int,
	timeSpent time.Duration,
) PerformedNetworkSearch {
	return PerformedNetworkSearch{
		AmountOfAskablePeers:       amountOfAskablePeers,
		AmountOfItemsAcrossAnswers: amountOfItemsAcrossAnswers(answers),
		AmountOfItemsInRanking:     len(rankedItems),
		AmountOfRankedItemsOfTheMostRankedPeer: amountOfRankedItemsOfTheMostRankedPeer(
			answers.ItemsInTheOrderOfEachPeerRanking,
			rankedItems,
		),
		AmountOfRankedItemsCountedByAPeer: amountOfItemsCountedByAPeer(rankedItems),
		TimeSpent:                         timeSpent,
	}
}

func amountOfItemsAcrossAnswers(answers queryanswers.AnsweredQuery) int {
	amountOfAnsweredItems := len(answers.ItemsInNoOrder)
	for _, itemsOfOnePeerRanking := range answers.ItemsInTheOrderOfEachPeerRanking {
		amountOfAnsweredItems += len(itemsOfOnePeerRanking)
	}

	return amountOfAnsweredItems
}

func amountOfRankedItemsOfTheMostRankedPeer(
	itemsInTheOrderOfEachPeerRanking [][]queryanswers.AnsweredItem,
	rankedItems []queryanswers.AnsweredItem,
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
	items []queryanswers.AnsweredItem,
	rankedDocuments map[yacymodel.URLHash]struct{},
) int {
	rankedDocumentsOfThePeer := make(map[yacymodel.URLHash]struct{}, len(items))
	for _, item := range items {
		if _, ranked := rankedDocuments[item.Metadata.Hash]; !ranked {
			continue
		}
		rankedDocumentsOfThePeer[item.Metadata.Hash] = struct{}{}
	}

	return len(rankedDocumentsOfThePeer)
}

func amountOfItemsCountedByAPeer(items []queryanswers.AnsweredItem) int {
	amount := 0
	for _, item := range items {
		if !item.CountedByAPeer() {
			continue
		}
		amount++
	}

	return amount
}
