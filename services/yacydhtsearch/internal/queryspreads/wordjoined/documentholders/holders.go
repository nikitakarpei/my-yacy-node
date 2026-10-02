// Package documentholders keeps which peers listed each document in their
// answers and orders documents by how many peers hold them.
package documentholders

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type Holders map[yacymodel.URLHash]map[yacymodel.Hash]struct{}

func (holders Holders) AddHoldersIn(answers []wordpartitionasks.ReplicaAnswer) {
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if holders[listedDocument.Hash] == nil {
				holders[listedDocument.Hash] = map[yacymodel.Hash]struct{}{}
			}
			holders[listedDocument.Hash][answer.Replica.Hash] = struct{}{}
		}
	}
}

func (holders Holders) MostHeldFirst(documents yacymodel.URLHashes) []yacymodel.URLHash {
	return slices.SortedFunc(
		maps.Keys(documents),
		func(first, second yacymodel.URLHash) int {
			if len(holders[first]) != len(holders[second]) {
				return cmp.Compare(len(holders[second]), len(holders[first]))
			}

			return strings.Compare(first.String(), second.String())
		},
	)
}
