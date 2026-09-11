package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	itemsInTheOrderOfEachAnswer [][]peeranswers.AnsweredItem,
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
	queryWords []yacymodel.Hash,
) peeranswers.AnsweredQuery {
	return peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachAnswer: itemsInTheOrderOfEachAnswerWithACountForEveryQueryWord(
			itemsInTheOrderOfEachAnswer, queryWords,
		),
		ItemsInNoOrder: itemsWithACountForEveryQueryWord(
			itemsOfAnsweredURLMetadataAsks(answeredURLMetadataAsks), queryWords,
		),
		DocumentsHeldPerQueryWord: documentsHeldPerQueryWordOf(answeredHeldDocumentsAsks),
	}
}

func itemsInTheOrderOfEachAnswerWithACountForEveryQueryWord(
	itemsInTheOrderOfEachAnswer [][]peeranswers.AnsweredItem,
	queryWords []yacymodel.Hash,
) [][]peeranswers.AnsweredItem {
	countedItemsInTheOrderOfEachAnswer := make(
		[][]peeranswers.AnsweredItem, 0, len(itemsInTheOrderOfEachAnswer),
	)
	for _, items := range itemsInTheOrderOfEachAnswer {
		countedItemsInTheOrderOfEachAnswer = append(
			countedItemsInTheOrderOfEachAnswer, itemsWithACountForEveryQueryWord(items, queryWords),
		)
	}

	return countedItemsInTheOrderOfEachAnswer
}

func itemsWithACountForEveryQueryWord(
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
		documentsHeldForTheWord, counted := answeredAsk.AmountOfDocumentsHeldForTheWord.Get()
		if !counted {
			continue
		}
		documentsHeldPerQueryWord[answeredAsk.Ask.Word] += documentsHeldForTheWord
	}

	return documentsHeldPerQueryWord
}
