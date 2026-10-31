package postingstree

import "crypto/sha256"

const (
	leafDomain byte = iota
	innerNodeDomain
	emptySubtreeDomain
)

type Node struct {
	hash     [sha256.Size]byte
	position Position
}

func LeafOf(encoding []byte, position Position) Node {
	return Node{
		hash:     sha256.Sum256(append([]byte{leafDomain}, encoding...)),
		position: position,
	}
}

func compareNodes(first, second Node) int {
	return first.position.compare(second.position)
}

func innerNodeHashOf(left, right [sha256.Size]byte, height uint8) [sha256.Size]byte {
	if left == emptySubtreeHashes[height-1] && right == emptySubtreeHashes[height-1] {
		return emptySubtreeHashes[height]
	}
	encoding := make([]byte, 0, 1+2*sha256.Size)
	encoding = append(encoding, innerNodeDomain)
	encoding = append(encoding, left[:]...)

	return sha256.Sum256(append(encoding, right[:]...))
}

var emptySubtreeHashes = emptySubtreeHashesByHeight()

func emptySubtreeHashesByHeight() [Depth + 1][sha256.Size]byte {
	var hashes [Depth + 1][sha256.Size]byte
	for height := range hashes {
		hashes[height] = sha256.Sum256([]byte{emptySubtreeDomain, byte(height)})
	}

	return hashes
}
