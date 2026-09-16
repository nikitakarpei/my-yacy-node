package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func itemsInTheOrderOfEachPeerRankingOf(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
	joinedDocuments map[yacymodel.URLHash]struct{},
) [][]queryanswers.AnsweredItem {
	itemsInTheOrderOfEachPeerRanking := make([][]queryanswers.AnsweredItem, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		items := itemsOfJoinedDocumentsAmong(answeredAsk, joinedDocuments)
		if len(items) == 0 {
			continue
		}
		itemsInTheOrderOfEachPeerRanking = append(itemsInTheOrderOfEachPeerRanking, items)
	}

	return itemsInTheOrderOfEachPeerRanking
}

func itemsOfJoinedDocumentsAmong(
	answeredAsk peerasks.AnsweredHeldDocumentsAsk,
	joinedDocuments map[yacymodel.URLHash]struct{},
) []queryanswers.AnsweredItem {
	keptItems := make([]queryanswers.AnsweredItem, 0, len(answeredAsk.MatchedDocuments))
	for _, matchedDocument := range answeredAsk.MatchedDocuments {
		if _, joined := joinedDocuments[matchedDocument.Metadata.Hash]; !joined {
			continue
		}
		keptItems = append(keptItems, queryanswers.AnsweredItem{
			Metadata: matchedDocument.Metadata,
			MatchedWords: map[yacymodel.Hash]queryanswers.WordCount{
				answeredAsk.Ask.Word: matchedDocument.CountOfAWordTheAskNamed,
			},
		})
	}

	return keptItems
}
