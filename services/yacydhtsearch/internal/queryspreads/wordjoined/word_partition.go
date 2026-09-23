package wordjoined

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type wordPartition struct {
	word      yacymodel.Hash
	partition uint
	replicas  []wordReplica
}

func (wordPartition wordPartition) hasACompleteAbstract() bool {
	return slices.ContainsFunc(wordPartition.replicas, wordReplica.hasACompleteAbstract)
}

func (wordPartition wordPartition) replicasWithoutACompleteAbstract() []wordReplica {
	var replicas []wordReplica
	for _, replica := range wordPartition.replicas {
		if replica.hasACompleteAbstract() {
			continue
		}
		replicas = append(replicas, replica)
	}

	return replicas
}
