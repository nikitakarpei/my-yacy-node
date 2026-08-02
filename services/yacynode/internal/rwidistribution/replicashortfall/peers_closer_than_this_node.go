package replicashortfall

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func peersCloserThanThisNode(
	peers []yacymodel.Seed,
	position yacymodel.DHTPosition,
	self yacymodel.Hash,
) []yacymodel.Hash {
	closer := make([]yacymodel.Hash, 0, len(peers))
	for _, seed := range peers {
		if yacymodel.CloserToPosition(seed.Hash, self, position) {
			closer = append(closer, seed.Hash)
		}
	}

	return closer
}
