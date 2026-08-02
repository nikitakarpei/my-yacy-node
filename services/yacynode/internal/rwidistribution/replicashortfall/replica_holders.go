package replicashortfall

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

// replicaHolders groups the peers holding a replica by the DHT responsibility
// that decides whether the ledger keeps their entry.
type replicaHolders struct {
	responsible   []yacymodel.Hash
	outsideWindow []yacymodel.Hash
	gone          []yacymodel.Hash
}

func (h replicaHolders) stillHolding() []yacymodel.Hash {
	return append(append([]yacymodel.Hash{}, h.responsible...), h.outsideWindow...)
}

func (r *Shortfall) holdersByResponsibility(
	ctx context.Context,
	holders []yacymodel.Hash,
	position yacymodel.DHTPosition,
	responsibilityWindow []yacymodel.Seed,
	replicasPeersOwe int,
) replicaHolders {
	var held replicaHolders
	for _, peer := range holders {
		switch {
		case !r.reachability.Reachable(ctx, peer):
			if r.reachability.RecentlyReachable(ctx, peer) {
				held.responsible = append(held.responsible, peer)
			} else {
				held.gone = append(held.gone, peer)
			}
		case noFartherThanClosestPeers(peer, position, responsibilityWindow, replicasPeersOwe):
			held.responsible = append(held.responsible, peer)
		default:
			held.outsideWindow = append(held.outsideWindow, peer)
		}
	}

	return held
}

func noFartherThanClosestPeers(
	peer yacymodel.Hash,
	position yacymodel.DHTPosition,
	closestPeers []yacymodel.Seed,
	replicasPeersOwe int,
) bool {
	if len(closestPeers) == 0 || len(closestPeers) < replicasPeersOwe {
		return true
	}
	farthest := closestPeers[len(closestPeers)-1].Hash

	return yacymodel.DistanceToPosition(peer, position) <=
		yacymodel.DistanceToPosition(farthest, position)
}
