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
