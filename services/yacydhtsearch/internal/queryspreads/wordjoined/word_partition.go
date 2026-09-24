package wordjoined

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

type wordPartition struct {
	word      yacymodel.Hash
	partition uint
	replicas  []wordReplica
}
