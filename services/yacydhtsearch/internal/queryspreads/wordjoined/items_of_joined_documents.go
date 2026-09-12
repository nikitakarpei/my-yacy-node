package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func itemsInTheOrderOfEachPeerRankingAmong(
	answeredAsks []peerasks.AnsweredHeldDocumentsAsk,
	joinedDocuments map[yacymodel.URLHash]struct{},
) [][]peeranswers.AnsweredItem {
	itemsInTheOrderOfEachPeerRanking := make([][]peeranswers.AnsweredItem, 0, len(answeredAsks))
	for _, answeredAsk := range answeredAsks {
		items := itemsOfJoinedDocumentsIn(answeredAsk, joinedDocuments)
		if len(items) == 0 {
			continue
		}
		itemsInTheOrderOfEachPeerRanking = append(itemsInTheOrderOfEachPeerRanking, items)
	}

	return itemsInTheOrderOfEachPeerRanking
}

func itemsOfJoinedDocumentsIn(
	answeredAsk peerasks.AnsweredHeldDocumentsAsk,
	joinedDocuments map[yacymodel.URLHash]struct{},
) []peeranswers.AnsweredItem {
	kept := make([]peeranswers.AnsweredItem, 0, len(answeredAsk.MatchedDocuments))
	for _, matchedDocument := range answeredAsk.MatchedDocuments {
		if _, joined := joinedDocuments[matchedDocument.Metadata.Hash]; !joined {
			continue
		}
		kept = append(kept, peeranswers.AnsweredItem{
			Metadata: matchedDocument.Metadata,
			MatchedWords: map[yacymodel.Hash]peeranswers.WordCount{
				answeredAsk.Ask.Word: matchedDocument.CountOfAWordTheAskNamed,
			},
		})
	}

	return kept
}
