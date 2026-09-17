package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	itemsInTheOrderOfEachPeerRanking [][]queryanswers.AnsweredItem,
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
	answeredHeldDocumentsAsks []peerasks.AnsweredHeldDocumentsAsk,
	queryWords []yacymodel.Hash,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:                       queryWords,
		ItemsInTheOrderOfEachPeerRanking: itemsInTheOrderOfEachPeerRanking,
		ItemsInNoOrder:                   itemsOfAnsweredURLMetadataAsks(answeredURLMetadataAsks),
		DocumentsHeldPerQueryWord:        documentsHeldPerQueryWordOf(answeredHeldDocumentsAsks),
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
