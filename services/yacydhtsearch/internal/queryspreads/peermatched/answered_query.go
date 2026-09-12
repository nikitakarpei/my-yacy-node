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
		QueryWords: queryWords,
		ItemsInTheOrderOfEachPeerRanking: itemsInTheOrderOfEachPeerRankingOf(
			answeredAsks,
			queryWords,
		),
	}
}

func itemsInTheOrderOfEachPeerRankingOf(
	answeredAsks []peerasks.AnsweredMatchedItemsAsk,
	queryWords []yacymodel.Hash,
) [][]peeranswers.AnsweredItem {
	itemsInTheOrderOfEachPeerRanking := make([][]peeranswers.AnsweredItem, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		itemsInTheOrderOfEachPeerRanking = append(
			itemsInTheOrderOfEachPeerRanking,
			itemsOfTheMatchedDocuments(answeredAsk.MatchedDocuments, queryWords),
		)
	}

	return itemsInTheOrderOfEachPeerRanking
}

func itemsOfTheMatchedDocuments(
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
	if len(queryWords) != 1 {
		return nil
	}

	return map[yacymodel.Hash]peeranswers.WordCount{
		queryWords[0]: matchedDocument.CountOfAWordTheAskNamed,
	}
}
