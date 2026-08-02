package postingoffer

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type replicaHolders struct {
	insideWindow  []yacymodel.Hash
	outsideWindow []yacymodel.Hash
	expired       []yacymodel.Hash
}

func (h replicaHolders) peersHoldingReplica(peers []yacymodel.Seed) []yacymodel.Seed {
	holdingPeers := make([]yacymodel.Seed, 0, len(h.insideWindow))
	for _, seed := range peers {
		if slices.Contains(h.insideWindow, seed.Hash) {
			holdingPeers = append(holdingPeers, seed)
		}
	}

	return holdingPeers
}

func (h replicaHolders) peersMissingReplica(peers []yacymodel.Seed) []yacymodel.Seed {
	unexpiredHolders := h.unexpiredHolders()

	missingPeers := make([]yacymodel.Seed, 0, len(peers))
	for _, seed := range peers {
		if !slices.Contains(unexpiredHolders, seed.Hash) {
			missingPeers = append(missingPeers, seed)
		}
	}

	return missingPeers
}

func (h replicaHolders) unexpiredHolders() []yacymodel.Hash {
	return slices.Concat(h.insideWindow, h.outsideWindow)
}
