package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartition struct {
	word     yacymodel.Hash
	replicas []wordReplica
}

func (wordPartition wordPartition) isFullyListed() bool {
	return slices.ContainsFunc(wordPartition.replicas, wordReplica.isFullyListed)
}

func (wordPartition wordPartition) replicasThatDidNotListAllTheyHold() []wordReplica {
	var replicas []wordReplica
	for _, replica := range wordPartition.replicas {
		if replica.isFullyListed() {
			continue
		}
		replicas = append(replicas, replica)
	}

	return replicas
}
