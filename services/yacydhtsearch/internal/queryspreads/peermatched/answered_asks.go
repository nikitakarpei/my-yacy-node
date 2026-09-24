package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
)

func answeredAsksFrom(
	settledWordPartitions <-chan replicaasks.SettledWordPartition,
) []peerasks.AnsweredSearchDocumentsAsk {
	askOutcomes := peerasks.SearchDocumentsAskOutcomes{}
	for settledWordPartition := range settledWordPartitions {
		askOutcomes = append(askOutcomes, settledWordPartition.AskOutcomes...)
	}

	return askOutcomes.AnsweredAsks()
}
