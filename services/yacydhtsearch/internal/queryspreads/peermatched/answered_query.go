package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	answeredAsks []peerasks.AnsweredMatchedItemsAsk,
	queryWords []yacymodel.Hash,
) peeranswers.AnsweredQuery {
	return peeranswers.AnsweredQuery{
		ItemsInTheOrderOfEachAnswer: itemsInTheOrderOfEachAnswerOf(answeredAsks, queryWords),
	}
}

func itemsInTheOrderOfEachAnswerOf(
	answeredAsks []peerasks.AnsweredMatchedItemsAsk,
	queryWords []yacymodel.Hash,
) [][]peeranswers.AnsweredItem {
	itemsInTheOrderOfEachAnswer := make([][]peeranswers.AnsweredItem, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		itemsInTheOrderOfEachAnswer = append(
			itemsInTheOrderOfEachAnswer,
			itemsWithACountForEveryQueryWordFrom(answeredAsk.MatchedDocuments, queryWords),
		)
	}

	return itemsInTheOrderOfEachAnswer
}

func itemsWithACountForEveryQueryWordFrom(
	matchedDocuments []peerasks.MatchedDocument,
	queryWords []yacymodel.Hash,
) []peeranswers.AnsweredItem {
	items := make([]peeranswers.AnsweredItem, 0, len(matchedDocuments))
	for _, matchedDocument := range matchedDocuments {
		items = append(items, peeranswers.AnsweredItem{
			Metadata:     matchedDocument.Metadata,
			MatchedWords: countPerQueryWordOf(matchedDocument, queryWords),
		})
	}

	return items
}

func countPerQueryWordOf(
	matchedDocument peerasks.MatchedDocument,
	queryWords []yacymodel.Hash,
) map[yacymodel.Hash]peeranswers.WordCount {
	countPerQueryWord := make(map[yacymodel.Hash]peeranswers.WordCount, len(queryWords))
	for _, queryWord := range queryWords {
		countPerQueryWord[queryWord] = peeranswers.WordCount{}
	}
	if len(queryWords) == 1 {
		countPerQueryWord[queryWords[0]] = matchedDocument.CountOfAWordTheAskNamed
	}

	return countPerQueryWord
}
