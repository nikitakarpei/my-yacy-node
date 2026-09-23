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
