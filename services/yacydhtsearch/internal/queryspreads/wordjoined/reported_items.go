package wordjoined

import (
	"maps"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func reportedItemsOfJoinedDocuments(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
	joinedDocuments map[yacymodel.URLHash]struct{},
) [][]searchresult.Item {
	itemsOfEachAnsweredAsk := make([][]searchresult.Item, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		items := itemsOfJoinedDocumentsAmong(answeredAsk.Items, joinedDocuments)
		if len(items) == 0 {
			continue
		}
		itemsOfEachAnsweredAsk = append(itemsOfEachAnsweredAsk, items)
	}

	return itemsOfEachAnsweredAsk
}

func itemsOfJoinedDocumentsAmong(
	items []searchresult.Item,
	joinedDocuments map[yacymodel.URLHash]struct{},
) []searchresult.Item {
	kept := make([]searchresult.Item, 0, len(items))
	for _, item := range items {
		if _, joined := joinedDocuments[item.Hash]; !joined {
			continue
		}
		kept = append(kept, item)
	}

	return kept
}

func joinedDocumentsWithoutAReportedItem(
	joinedDocuments map[yacymodel.URLHash]struct{},
	reportedItems [][]searchresult.Item,
) map[yacymodel.URLHash]struct{} {
	withoutAnItem := make(map[yacymodel.URLHash]struct{}, len(joinedDocuments))
	maps.Copy(withoutAnItem, joinedDocuments)
	for _, items := range reportedItems {
		for _, item := range items {
			delete(withoutAnItem, item.Hash)
		}
	}

	return withoutAnItem
}
