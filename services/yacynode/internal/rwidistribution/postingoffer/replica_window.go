package postingoffer

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type replicaWindow struct {
	closest          []yacymodel.Seed
	position         yacymodel.DHTPosition
	requiredReplicas int
}

func replicaWindowFor(
	position yacymodel.DHTPosition,
	acceptingPeers []yacymodel.Seed,
	self yacymodel.Hash,
	redundancy int,
) replicaWindow {
	requiredReplicas := redundancy
	if len(yacymodel.SeedsCloserThanPeer(acceptingPeers, self, position)) < redundancy {
		requiredReplicas = redundancy - 1
	}

	return replicaWindow{
		closest: yacymodel.SeedsClosestToPosition(
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

	return yacymodel.DistanceToPosition(peer, r.position) <=
		yacymodel.DistanceToPosition(farthest, r.position)
}
