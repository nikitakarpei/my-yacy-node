package replicashortfall

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func peersWithoutReplica(peers []yacymodel.Seed, holders []yacymodel.Hash) []yacymodel.Seed {
	without := make([]yacymodel.Seed, 0, len(peers))
	for _, seed := range peers {
		if !slices.Contains(holders, seed.Hash) {
			without = append(without, seed)
		}
	}

	return without
}
