package replicashortfall

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func peersHoldingReplica(peers []yacymodel.Seed, holders []yacymodel.Hash) []yacymodel.Seed {
	holding := make([]yacymodel.Seed, 0, len(holders))
	for _, seed := range peers {
		if slices.Contains(holders, seed.Hash) {
			holding = append(holding, seed)
		}
	}

	return holding
}
