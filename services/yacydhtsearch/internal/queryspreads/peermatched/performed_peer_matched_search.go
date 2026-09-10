package peermatched

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedPeerMatchedSearch struct {
	AmountOfQueryWords              int
	AmountOfAskedPeers              int
	AmountOfPeersThatAnswered       int
	AmountOfPeersThatMatchedNothing int
	TimeSpent                       time.Duration
}

func performedPeerMatchedSearchFrom(
	queryWords []yacymodel.Hash,
	asks []peerasks.MatchedItemsAsk,
	answeredAsks []peerasks.AnsweredMatchedItemsAsk,
	timeSpent time.Duration,
) PerformedPeerMatchedSearch {
	return PerformedPeerMatchedSearch{
		AmountOfQueryWords:              len(queryWords),
		AmountOfAskedPeers:              len(asks),
		AmountOfPeersThatAnswered:       len(answeredAsks),
		AmountOfPeersThatMatchedNothing: amountOfPeersThatMatchedNothing(answeredAsks),
		TimeSpent:                       timeSpent,
	}
}

func amountOfPeersThatMatchedNothing(answeredAsks []peerasks.AnsweredMatchedItemsAsk) int {
	var peersThatMatchedNothing int
	for _, answeredAsk := range answeredAsks {
		if len(answeredAsk.Items) != 0 {
			continue
		}
		peersThatMatchedNothing++
	}

	return peersThatMatchedNothing
}
