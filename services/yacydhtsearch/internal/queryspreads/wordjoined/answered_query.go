package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
)

func answeredQueryFrom(
	matchedAndHeldDocumentsRound matchedAndHeldDocumentsRound,
	joinOfTheQuery joinOfTheQuery,
	urlMetadataRound urlMetadataRound,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords: matchedAndHeldDocumentsRound.queryWords,
		ItemsInTheOrderOfEachPeerRanking: itemsInTheOrderOfEachPeerRankingOf(
			matchedAndHeldDocumentsRound.answeredAsks,
			joinOfTheQuery.documentsJoinedWithCrossChecking,
		),
		ItemsInNoOrder: itemsOfAnsweredURLMetadataAsks(urlMetadataRound.answeredAsks),
		DocumentsHeldPerQueryWord: matchedAndHeldDocumentsRound.
			amountOfDocumentsHeldPerQueryWord(),
	}
}

func itemsOfAnsweredURLMetadataAsks(
	answeredAsks []peerasks.AnsweredURLMetadataAsk,
) []queryanswers.AnsweredItem {
	var items []queryanswers.AnsweredItem
	for _, answeredAsk := range answeredAsks {
		for _, metadata := range answeredAsk.MetadataOfEachDocument {
			items = append(items, queryanswers.AnsweredItem{Metadata: metadata})
		}
	}

	return items
}
