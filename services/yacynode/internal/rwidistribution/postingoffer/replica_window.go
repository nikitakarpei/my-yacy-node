package postingoffer

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type replicaWindow struct {
	farthestPeer     yacymodel.Hash
	coversWholeRing  bool
	position         yacymodel.DHTRingPosition
	requiredReplicas int
}

func replicaWindowFor(
	position yacymodel.DHTRingPosition,
	acceptingPeers yacymodel.SeedRing,
	self yacymodel.Hash,
	redundancy int,
) replicaWindow {
	closest := acceptingPeers.SeedsClosestTo(position, redundancy)
	requiredReplicas := redundancy
	if len(closest) < redundancy ||
		!yacymodel.CloserToDHTRingPosition(closest[redundancy-1].Hash, self, position) {
		requiredReplicas = redundancy - 1
	}
	if requiredReplicas == 0 || len(closest) < requiredReplicas {
		return replicaWindow{
			coversWholeRing:  true,
			position:         position,
			requiredReplicas: requiredReplicas,
		}
	}

	return replicaWindow{
		farthestPeer:     closest[requiredReplicas-1].Hash,
		position:         position,
		requiredReplicas: requiredReplicas,
	}
}

func (r replicaWindow) contains(peer yacymodel.Hash) bool {
	return r.coversWholeRing ||
		!yacymodel.CloserToDHTRingPosition(r.farthestPeer, peer, r.position)
}
