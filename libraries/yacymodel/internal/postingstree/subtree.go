package postingstree

import (
	"crypto/sha256"
	"slices"
	"sort"
)

type Subtree struct {
	leaves   []Node
	entries  []Node
	position Position
}

func SubtreeOf(leaves, entries []Node) Subtree {
	sortedLeaves := slices.Clone(leaves)
	slices.SortFunc(sortedLeaves, compareNodes)

	return Subtree{leaves: sortedLeaves, entries: entries}
}

func (s Subtree) Hash() [sha256.Size]byte {
	switch {
	case len(s.leaves) > 0 && s.position.depth == Depth:
		return s.leaves[0].hash
	case len(s.leaves) > 0:
		return s.hashOfChildren()
	case len(s.entries) > 0 && s.entries[0].position == s.position:
		return s.entries[0].hash
	case len(s.entries) > 0:
		return s.hashOfChildren()
	default:
		return emptySubtreeHashes[s.position.height()]
	}
}

func (s Subtree) hashOfChildren() [sha256.Size]byte {
	left, right := s.children()

	return innerNodeHashOf(left.Hash(), right.Hash(), s.position.height())
}

func (s Subtree) children() (Subtree, Subtree) {
	entriesBelow := s.entries
	if len(entriesBelow) > 0 && entriesBelow[0].position == s.position {
		entriesBelow = entriesBelow[1:]
	}
	leftLeaves, rightLeaves := splitAt(s.position.depth, s.leaves, positionOfNode)
	leftEntries, rightEntries := splitAt(s.position.depth, entriesBelow, positionOfNode)
	left := Subtree{
		leaves:   leftLeaves,
		entries:  leftEntries,
		position: s.position.childAt(0),
	}
	right := Subtree{
		leaves:   rightLeaves,
		entries:  rightEntries,
		position: s.position.childAt(1),
	}

	return left, right
}

func splitAt[T any](depth uint8, sorted []T, positionOf func(T) Position) (left, right []T) {
	rightStart := sort.Search(len(sorted), func(i int) bool {
		return positionOf(sorted[i]).bitAt(depth) == 1
	})

	return sorted[:rightStart], sorted[rightStart:]
}

func positionOfNode(node Node) Position {
	return node.position
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

func (s Subtree) EntriesAround(keptPositions []Position) []Node {
	return s.entriesAround(sortedPositionsOf(keptPositions))
}

func (s Subtree) entriesAround(keptPositions []Position) []Node {
	if len(keptPositions) == 0 {
		return s.summaryEntries()
	}
	if s.position.depth == Depth {
		return nil
	}
	left, right := s.children()
	keptLeft, keptRight := splitAt(s.position.depth, keptPositions, positionItself)

	return append(left.entriesAround(keptLeft), right.entriesAround(keptRight)...)
}

func (s Subtree) summaryEntries() []Node {
	hash := s.Hash()
	if hash == emptySubtreeHashes[s.position.height()] {
		return nil
	}

	return []Node{{hash: hash, position: s.position}}
}

func positionItself(position Position) Position {
	return position
}
