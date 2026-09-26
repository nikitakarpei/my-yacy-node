package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

func answeredAsksFrom(
	settledWordPartitions <-chan wordpartitionasks.SettledWordPartition,
) []peerasks.AnsweredSearchDocumentsAsk {
	askOutcomes := peerasks.SearchDocumentsAskOutcomes{}
	for settledWordPartition := range settledWordPartitions {
		askOutcomes = append(askOutcomes, settledWordPartition.AskOutcomes...)
	}

	return askOutcomes.AnsweredAsks()
}
