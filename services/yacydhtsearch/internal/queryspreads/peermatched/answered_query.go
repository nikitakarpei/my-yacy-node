package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords: queryWords,
		ItemsInTheOrderOfEachPeerRanking: itemsInTheOrderOfEachPeerRankingOf(
			answeredAsks,
			queryWords,
		),
	}
}

func itemsInTheOrderOfEachPeerRankingOf(
	answeredAsks []peerasks.AnsweredMatchedDocumentsAsk,
	queryWords []yacymodel.Hash,
) [][]queryanswers.AnsweredItem {
	itemsInTheOrderOfEachPeerRanking := make([][]queryanswers.AnsweredItem, 0, len(answeredAsks))
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
) []queryanswers.AnsweredItem {
	items := make([]queryanswers.AnsweredItem, 0, len(matchedDocuments))
	for _, matchedDocument := range matchedDocuments {
		items = append(items, queryanswers.AnsweredItem{
			Metadata:     matchedDocument.Metadata,
			MatchedWords: countPerQueryWordOf(matchedDocument, queryWords),
		})
	}

	return items
}

func countPerQueryWordOf(
	matchedDocument peerasks.MatchedDocument,
	queryWords []yacymodel.Hash,
) map[yacymodel.Hash]queryanswers.WordCount {
	if len(queryWords) != 1 {
		return nil
	}

	return map[yacymodel.Hash]queryanswers.WordCount{
		queryWords[0]: matchedDocument.CountOfAWordTheAskNamed,
	}
}
