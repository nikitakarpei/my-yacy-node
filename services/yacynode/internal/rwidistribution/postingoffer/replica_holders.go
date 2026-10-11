package postingoffer

import (
	"iter"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type replicaHolders struct {
	insideWindow  []yacymodel.Hash
	outsideWindow []yacymodel.Hash
	expired       []yacymodel.Hash
}

func (h replicaHolders) peersHoldingReplica(peers iter.Seq[yacymodel.Seed]) []yacymodel.Seed {
	holdingPeers := make([]yacymodel.Seed, 0, len(h.insideWindow))
	for seed := range peers {
		if slices.Contains(h.insideWindow, seed.Hash) {
			holdingPeers = append(holdingPeers, seed)
		}
	}

	return holdingPeers
}

func (h replicaHolders) closestPeersMissingReplica(
	peers yacymodel.SeedRing,
	position yacymodel.DHTRingPosition,
	want int,
) []yacymodel.Seed {
	unexpiredHolders := h.unexpiredHolders()

	closestPeers := make([]yacymodel.Seed, 0, want)
	for seed := range peers.SeedsClockwiseFrom(position) {
		if len(closestPeers) == want {
			break
		}
		if !slices.Contains(unexpiredHolders, seed.Hash) {
			closestPeers = append(closestPeers, seed)
		}
	}

	return closestPeers
}

func (h replicaHolders) unexpiredHolders() []yacymodel.Hash {
	return slices.Concat(h.insideWindow, h.outsideWindow)
}
