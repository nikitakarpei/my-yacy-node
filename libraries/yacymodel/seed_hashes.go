package yacymodel

func PeerHashesOf(seeds []Seed) []Hash {
	peerHashes := make([]Hash, len(seeds))
	for index, seed := range seeds {
		peerHashes[index] = seed.Hash
	}

	return peerHashes
}
