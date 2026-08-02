package replicashortfall

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func closerToPositionThanThisNode(
	peer yacymodel.Hash,
	position yacymodel.DHTPosition,
	self yacymodel.Hash,
) bool {
	return yacymodel.DistanceToPosition(peer, position) <
		yacymodel.DistanceToPosition(self, position)
}

func peersCloserThanThisNode(
	peers []yacymodel.Seed,
	position yacymodel.DHTPosition,
	self yacymodel.Hash,
) []yacymodel.Seed {
	closer := make([]yacymodel.Seed, 0, len(peers))
	for _, seed := range peers {
		if closerToPositionThanThisNode(seed.Hash, position, self) {
			closer = append(closer, seed)
		}
	}

	return closer
}
