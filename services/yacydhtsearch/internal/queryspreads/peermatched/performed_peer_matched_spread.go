package peermatched

import (
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedPeerMatchedSpread struct {
	AmountOfQueryWords              int
	AmountOfPeersThatMatchedNothing int
	TimeSpent                       time.Duration
}

func performedPeerMatchedSpreadFrom(
	queryWords []yacymodel.Hash,
	amountOfPeersThatMatchedNothing int,
	timeSpent time.Duration,
) PerformedPeerMatchedSpread {
	return PerformedPeerMatchedSpread{
		AmountOfQueryWords:              len(queryWords),
		AmountOfPeersThatMatchedNothing: amountOfPeersThatMatchedNothing,
		TimeSpent:                       timeSpent,
	}
}
