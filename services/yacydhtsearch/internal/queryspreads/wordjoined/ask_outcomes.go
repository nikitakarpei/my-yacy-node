package wordjoined

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
)

func askOutcomesFrom(
	settledWordPartitions <-chan replicaasks.SettledWordPartition,
) peerasks.SearchDocumentsAskOutcomes {
	askOutcomes := peerasks.SearchDocumentsAskOutcomes{}
	for settledWordPartition := range settledWordPartitions {
		askOutcomes = append(askOutcomes, settledWordPartition.AskOutcomes...)
	}

	return askOutcomes
}
