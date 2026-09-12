package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	itemsInTheOrderOfEachPeerRanking [][]peeranswers.AnsweredItem,
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
	queryWords []yacymodel.Hash,
) peeranswers.AnsweredQuery {
	return peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachPeerRanking: itemsInTheOrderOfEachPeerRankingWithACountForEveryQueryWordFrom(
			itemsInTheOrderOfEachPeerRanking,
			queryWords,
		),
		ItemsInNoOrder: itemsWithACountForEveryQueryWordFrom(
			itemsOfAnsweredURLMetadataAsks(answeredURLMetadataAsks), queryWords,
		),
		DocumentsHeldPerQueryWord: documentsHeldPerQueryWordOf(answeredHeldDocumentsAsks),
	}
}

func itemsInTheOrderOfEachPeerRankingWithACountForEveryQueryWordFrom(
	itemsInTheOrderOfEachPeerRanking [][]peeranswers.AnsweredItem,
	queryWords []yacymodel.Hash,
) [][]peeranswers.AnsweredItem {
	countedItemsInTheOrderOfEachPeerRanking := make(
		[][]peeranswers.AnsweredItem, 0, len(itemsInTheOrderOfEachPeerRanking),
	)
	for _, items := range itemsInTheOrderOfEachPeerRanking {
		countedItemsInTheOrderOfEachPeerRanking = append(
			countedItemsInTheOrderOfEachPeerRanking,
			itemsWithACountForEveryQueryWordFrom(items, queryWords),
		)
	}

	return countedItemsInTheOrderOfEachPeerRanking
}

func itemsWithACountForEveryQueryWordFrom(
	items []peeranswers.AnsweredItem,
	queryWords []yacymodel.Hash,
) []peeranswers.AnsweredItem {
	countedItems := make([]peeranswers.AnsweredItem, 0, len(items))
	for _, item := range items {
		countedItems = append(countedItems, item.MatchingTheWords(queryWords))
	}

	return countedItems
}

func itemsOfAnsweredURLMetadataAsks(
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) []peeranswers.AnsweredItem {
	var items []peeranswers.AnsweredItem
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			items = append(items, peeranswers.AnsweredItem{Metadata: metadata})
		}
	}

	return items
}

func documentsHeldPerQueryWordOf(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
) map[yacymodel.Hash]int {
	documentsHeldPerQueryWord := make(map[yacymodel.Hash]int, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		amountOfDocumentsHeldForTheWord, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
		if !counted {
			continue
		}
		documentsHeldPerQueryWord[answeredAsk.Ask.Word] += amountOfDocumentsHeldForTheWord
	}

	return documentsHeldPerQueryWord
}
