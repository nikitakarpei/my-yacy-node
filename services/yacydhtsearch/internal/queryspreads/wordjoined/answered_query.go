package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func answeredQueryFrom(
	itemsInTheOrderOfEachPeerRanking [][]queryanswers.AnsweredItem,
	answeredURLMetadataAsks []peerasks.AnsweredURLMetadataAsk,
	amountOfDocumentsHeldPerQueryWord map[yacymodel.Hash]int,
	queryWords []yacymodel.Hash,
) queryanswers.AnsweredQuery {
	return queryanswers.AnsweredQuery{
		QueryWords:                       queryWords,
		ItemsInTheOrderOfEachPeerRanking: itemsInTheOrderOfEachPeerRanking,
		ItemsInNoOrder:                   itemsOfAnsweredURLMetadataAsks(answeredURLMetadataAsks),
		DocumentsHeldPerQueryWord:        amountOfDocumentsHeldPerQueryWord,
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
