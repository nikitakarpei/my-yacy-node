package documentrelevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const placesHalvingAPlaceScore = 61.0

func placeScorePerDocumentOf(
	itemsInTheOrderOfEachPeerRanking [][]peeranswers.AnsweredItem,
) map[yacymodel.URLHash]float64 {
	placeScorePerDocument := map[yacymodel.URLHash]float64{}
	for _, itemsOfOnePeerRanking := range itemsInTheOrderOfEachPeerRanking {
		for place, item := range itemsOfOnePeerRanking {
			placeScorePerDocument[item.Metadata.Hash] +=
				1 / (1 + float64(place)/placesHalvingAPlaceScore)
		}
	}

	return placeScorePerDocument
}
