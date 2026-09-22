package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartition struct {
	word     yacymodel.Hash
	replicas []queryWordOnReplica
}

func (partition wordPartition) isFullyListed() bool {
	return slices.ContainsFunc(partition.replicas, queryWordOnReplica.isFullyListed)
}

func (partition wordPartition) replicasThatDidNotListAllTheyHold() []queryWordOnReplica {
	var replicas []queryWordOnReplica
	for _, replica := range partition.replicas {
		if replica.isFullyListed() {
			continue
		}
		replicas = append(replicas, replica)
	}

	return replicas
}
