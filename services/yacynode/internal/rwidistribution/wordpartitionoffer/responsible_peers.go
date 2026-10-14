package wordpartitionoffer

import (
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type responsiblePeers struct {
	hashes           []yacymodel.Hash
	coversWholeRing  bool
	position         yacymodel.DHTRingPosition
	requiredReplicas int
}

func responsiblePeersFor(
	position yacymodel.DHTRingPosition,
	acceptingPeers yacymodel.SeedRing,
	self yacymodel.Hash,
	redundancy int,
) responsiblePeers {
	closest := acceptingPeers.SeedsClosestTo(position, redundancy)
	requiredReplicas := redundancy
	if len(closest) < redundancy ||
		!yacymodel.CloserToDHTRingPosition(closest[redundancy-1].Hash, self, position) {
		requiredReplicas = redundancy - 1
	}

	return responsiblePeers{
		hashes:           yacymodel.PeerHashesOf(closest[:min(requiredReplicas, len(closest))]),
		coversWholeRing:  requiredReplicas == 0 || len(closest) < requiredReplicas,
		position:         position,
		requiredReplicas: requiredReplicas,
	}
}

func (r responsiblePeers) contains(peer yacymodel.Hash) bool {
	return r.coversWholeRing ||
		!yacymodel.CloserToDHTRingPosition(r.farthestPeer(), peer, r.position)
}

func (r responsiblePeers) farthestPeer() yacymodel.Hash {
	return r.hashes[len(r.hashes)-1]
}
