package peermatched

import (
	"slices"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PerformedPeerMatchedSpread struct {
	AmountOfQueryWords              int
	AmountOfPeersThatMatchedNothing int
	TimeSpent                       time.Duration
}

func performedPeerMatchedSpreadFrom(
	queryWords []yacymodel.Hash,
	settledAsks []wordpartitionasks.SettledAsk,
	timeSpent time.Duration,
) PerformedPeerMatchedSpread {
	return PerformedPeerMatchedSpread{
		AmountOfQueryWords:              len(queryWords),
		AmountOfPeersThatMatchedNothing: amountOfPeersThatMatchedNothing(settledAsks),
		TimeSpent:                       timeSpent,
	}
}

func amountOfPeersThatMatchedNothing(settledAsks []wordpartitionasks.SettledAsk) int {
	var peersThatMatchedNothing int
	for _, settledAsk := range settledAsks {
		for _, answer := range settledAsk.Answers {
			if slices.ContainsFunc(answer.ListedDocuments, isMatched) {
				continue
			}
			peersThatMatchedNothing++
		}
	}

	return peersThatMatchedNothing
}

func isMatched(listedDocument wordpartitionasks.ListedDocument) bool {
	return listedDocument.Metadata.Present()
}
