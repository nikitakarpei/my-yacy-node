package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartition struct {
	word     yacymodel.Hash
	replicas []queryWordOnReplica
}

func (wordPartition wordPartition) isFullyListed() bool {
	return slices.ContainsFunc(wordPartition.replicas, queryWordOnReplica.isFullyListed)
}

func (wordPartition wordPartition) replicasThatDidNotListAllTheyHold() []queryWordOnReplica {
	var replicas []queryWordOnReplica
	for _, replica := range wordPartition.replicas {
		if replica.isFullyListed() {
			continue
		}
		replicas = append(replicas, replica)
	}

	return replicas
}
