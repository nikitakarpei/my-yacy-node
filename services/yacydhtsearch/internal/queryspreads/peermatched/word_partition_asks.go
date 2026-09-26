package peermatched

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
)

func wordPartitionAsksFor(
	query searchquery.Query,
	chosenPeersPerQueryWord peerchoice.ChosenPeersPerQueryWord,
) []wordpartitionasks.Ask {
	var asks []wordpartitionasks.Ask
	for _, chosenPeersOfQueryWord := range chosenPeersPerQueryWord {
		for _, chosenPeersOfPartition := range chosenPeersOfQueryWord.ChosenPeersPerPartition() {
			asks = append(asks, wordpartitionasks.Ask{
				Word:            chosenPeersOfQueryWord.QueryWord,
				Partition:       chosenPeersOfPartition.Partition,
				ReplicasInOrder: chosenPeersOfPartition.Peers,
				ExcludedWords:   query.ExclusionHashes(),
				Language:        query.Language,
			})
		}
	}

	return asks
}
