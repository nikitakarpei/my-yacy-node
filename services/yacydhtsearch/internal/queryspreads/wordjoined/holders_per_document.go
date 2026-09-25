package wordjoined

import (
	"cmp"
	"maps"
	"slices"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type holdersPerDocument map[yacymodel.URLHash]map[yacymodel.Hash]struct{}

func (holders holdersPerDocument) addHoldersIn(
	askOutcomes peerasks.WordAbstractAskOutcomes,
) {
	for _, answeredAsk := range askOutcomes.AnsweredAsks() {
		for _, document := range answeredAsk.Abstract {
			if holders[document] == nil {
				holders[document] = map[yacymodel.Hash]struct{}{}
			}
			holders[document][answeredAsk.Ask.Peer.Hash] = struct{}{}
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
