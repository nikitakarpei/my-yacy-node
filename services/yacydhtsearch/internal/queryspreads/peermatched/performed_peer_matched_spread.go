package peermatched

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedPeerMatchedSpread struct {
	AmountOfQueryWords              int
	AmountOfPeersThatMatchedNothing int
	TimeSpent                       time.Duration
}

func performedPeerMatchedSpreadFrom(
	queryWords []yacymodel.Hash,
	answeredAsks []peerasks.AnsweredSearchDocumentsAsk,
	timeSpent time.Duration,
) PerformedPeerMatchedSpread {
	return PerformedPeerMatchedSpread{
		AmountOfQueryWords:              len(queryWords),
		AmountOfPeersThatMatchedNothing: amountOfPeersThatMatchedNothing(answeredAsks),
		TimeSpent:                       timeSpent,
	}
}

func amountOfPeersThatMatchedNothing(answeredAsks []peerasks.AnsweredSearchDocumentsAsk) int {
	var peersThatMatchedNothing int
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.MatchedDocuments) != 0 {
			continue
		}
		peersThatMatchedNothing++
	}

	return peersThatMatchedNothing
}
