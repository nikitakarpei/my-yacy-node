package networksearch

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedNetworkSearch struct {
	AmountOfAskablePeers            int
	AmountOfItemsAcrossAnswers      int
	AmountOfItemsInRanking          int
	AmountOfRankedItemsOfTheOnePeer int
	TimeSpent                       time.Duration
}

func performedNetworkSearchFrom(
	itemsOfEachPeer [][]searchresult.Item,
	ranking searchresult.Ranking,
	amountOfAskablePeers int,
	timeSpent time.Duration,
) PerformedNetworkSearch {
	return PerformedNetworkSearch{
		AmountOfAskablePeers:       amountOfAskablePeers,
		AmountOfItemsAcrossAnswers: amountOfItemsAcrossAnswers(itemsOfEachPeer),
		AmountOfItemsInRanking:     len(ranking.Items),
		AmountOfRankedItemsOfTheOnePeer: amountOfRankedItemsOfTheOnePeer(
			itemsOfEachPeer,
			ranking,
		),
		TimeSpent: timeSpent,
	}
}

func amountOfItemsAcrossAnswers(itemsOfEachPeer [][]searchresult.Item) int {
	var amountOfAnsweredItems int
	for _, items := range itemsOfEachPeer {
		amountOfAnsweredItems += len(items)
	}

	return amountOfAnsweredItems
}

func amountOfRankedItemsOfTheOnePeer(
	itemsOfEachPeer [][]searchresult.Item,
	ranking searchresult.Ranking,
) int {
	rankedDocuments := make(map[yacymodel.URLHash]struct{}, len(ranking.Items))
	for _, item := range ranking.Items {
		rankedDocuments[item.Hash] = struct{}{}
	}

	var mostRankedItemsOfOnePeer int
	for _, items := range itemsOfEachPeer {
		mostRankedItemsOfOnePeer = max(
			mostRankedItemsOfOnePeer,
			amountOfRankedItemsAmong(items, rankedDocuments),
		)
	}

	return mostRankedItemsOfOnePeer
}

func amountOfRankedItemsAmong(
	items []searchresult.Item,
	rankedDocuments map[yacymodel.URLHash]struct{},
) int {
	countedDocuments := make(map[yacymodel.URLHash]struct{}, len(items))
	for _, item := range items {
		if _, ranked := rankedDocuments[item.Hash]; !ranked {
			continue
		}
		countedDocuments[item.Hash] = struct{}{}
	}

	return len(countedDocuments)
}
