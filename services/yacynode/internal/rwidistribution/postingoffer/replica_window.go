package postingoffer

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type replicaWindow struct {
	closest          []yacymodel.Seed
	position         yacymodel.DHTRingPosition
	requiredReplicas int
}

func replicaWindowFor(
	position yacymodel.DHTRingPosition,
	acceptingPeers []yacymodel.Seed,
	self yacymodel.Hash,
	redundancy int,
) replicaWindow {
	requiredReplicas := redundancy
	if len(
		yacymodel.SeedsCloserToDHTRingPositionThanPeer(acceptingPeers, self, position),
	) < redundancy {
		requiredReplicas = redundancy - 1
	}

	return replicaWindow{
		closest: yacymodel.SeedsClosestToDHTRingPosition(
			acceptingPeers, position, requiredReplicas,
		),
		position:         position,
		requiredReplicas: requiredReplicas,
	}
}

func (r replicaWindow) contains(peer yacymodel.Hash) bool {
	if len(r.closest) < max(r.requiredReplicas, 1) {
		return true
	}
	farthest := r.closest[len(r.closest)-1].Hash

	return r.position.DistanceTo(yacymodel.DHTRingPositionOf(peer)) <=
		r.position.DistanceTo(yacymodel.DHTRingPositionOf(farthest))
}
