package relevance

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const placesHalvingAPlaceScore = 61.0

func placeScorePerDocumentOf(
	itemsInTheOrderOfEachAnswer [][]peeranswers.AnsweredItem,
) map[yacymodel.URLHash]float64 {
	placeScorePerDocument := map[yacymodel.URLHash]float64{}
	for _, itemsOfOneAnswer := range itemsInTheOrderOfEachAnswer {
		for place, item := range itemsOfOneAnswer {
			placeScorePerDocument[item.Metadata.Hash] +=
				1 / (1 + float64(place)/placesHalvingAPlaceScore)
		}
	}

	return placeScorePerDocument
}
