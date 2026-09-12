package peermatched

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedPeerMatchedSpread struct {
	AmountOfQueryWords              int
	AmountOfPeersAsked              int
	AmountOfPeersThatAnswered       int
	AmountOfPeersThatMatchedNothing int
	TimeSpent                       time.Duration
}

func performedPeerMatchedSpreadFrom(
	queryWords []yacymodel.Hash,
	asks []peerasks.MatchedItemsAsk,
	answeredAsks []peerasks.AnsweredMatchedItemsAsk,
	timeSpent time.Duration,
) PerformedPeerMatchedSpread {
	return PerformedPeerMatchedSpread{
		AmountOfQueryWords:              len(queryWords),
		AmountOfPeersAsked:              len(asks),
		AmountOfPeersThatAnswered:       len(answeredAsks),
		AmountOfPeersThatMatchedNothing: amountOfPeersThatMatchedNothing(answeredAsks),
		TimeSpent:                       timeSpent,
	}
}

func amountOfPeersThatMatchedNothing(answeredAsks []peerasks.AnsweredMatchedItemsAsk) int {
	var peersThatMatchedNothing int
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.MatchedDocuments) != 0 {
			continue
		}
		peersThatMatchedNothing++
	}

	return peersThatMatchedNothing
}
