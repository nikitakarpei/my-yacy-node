package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
)

func answeredAsksFrom(
	settledWordPartitions <-chan replicaasks.SettledWordPartition,
	amountOfAsks int,
) []peerasks.AnsweredSearchDocumentsAsk {
	askOutcomes := make(peerasks.SearchDocumentsAskOutcomes, amountOfAsks)
	for settledWordPartition := range settledWordPartitions {
		for _, placed := range settledWordPartition.AskOutcomes {
			askOutcomes[placed.PlaceInTheRun] = placed.AskOutcome
		}
	}

	return askOutcomes.AnsweredAsks()
}
