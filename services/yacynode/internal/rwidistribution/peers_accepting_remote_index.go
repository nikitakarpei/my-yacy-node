package rwidistribution

import "github.com/nikitakarpei/yacy-rwi-node/yacymodel"

func acceptsRemoteIndex(seed yacymodel.Seed) bool {
	capabilities, known := seed.Capabilities.Get()

	return known && capabilities.AcceptRemoteIndex
}

func peersAcceptingRemoteIndex(peers []yacymodel.Seed) []yacymodel.Seed {
	accepting := make([]yacymodel.Seed, 0, len(peers))
	for _, seed := range peers {
		if acceptsRemoteIndex(seed) {
			accepting = append(accepting, seed)
		}
	}

	return accepting
}
