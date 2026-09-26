package wordjoined

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type holdersPerDocument map[yacymodel.URLHash]map[yacymodel.Hash]struct{}

func (holders holdersPerDocument) addHoldersIn(answers []wordpartitionasks.ReplicaAnswer) {
	for _, answer := range answers {
		for _, listedDocument := range answer.ListedDocuments {
			if holders[listedDocument.Hash] == nil {
				holders[listedDocument.Hash] = map[yacymodel.Hash]struct{}{}
			}
			holders[listedDocument.Hash][answer.Replica.Hash] = struct{}{}
		}
	}
}

func (holders holdersPerDocument) mostHeldFirst(documents distinctDocuments) []yacymodel.URLHash {
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
