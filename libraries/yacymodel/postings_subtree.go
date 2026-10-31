package yacymodel

import (
	"crypto/sha256"
	"sort"
)

type postingsSubtree struct {
	leaves   []postingsTreeNode
	entries  []postingsTreeNode
	position postingsTreePosition
}

func (s postingsSubtree) hash() [sha256.Size]byte {
	switch {
	case len(s.leaves) > 0 && s.position.depth == postingsTreeDepth:
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

func (s postingsSubtree) hashOfChildren() [sha256.Size]byte {
	left, right := s.children()

	return innerNodeHashOf(left.hash(), right.hash(), s.position.height())
}

func (s postingsSubtree) children() (postingsSubtree, postingsSubtree) {
	entriesBelow := s.entries
	if len(entriesBelow) > 0 && entriesBelow[0].position == s.position {
		entriesBelow = entriesBelow[1:]
	}
	leftLeaves, rightLeaves := splitAt(s.position.depth, s.leaves, positionOfNode)
	leftEntries, rightEntries := splitAt(s.position.depth, entriesBelow, positionOfNode)
	left := postingsSubtree{
		leaves:   leftLeaves,
		entries:  leftEntries,
		position: s.position.childAt(0),
	}
	right := postingsSubtree{
		leaves:   rightLeaves,
		entries:  rightEntries,
		position: s.position.childAt(1),
	}

	return left, right
}

func splitAt[T any](
	depth uint8,
	sorted []T,
	positionOf func(T) postingsTreePosition,
) (left, right []T) {
	rightStart := sort.Search(len(sorted), func(i int) bool {
		return positionOf(sorted[i]).bitAt(depth) == 1
	})

	return sorted[:rightStart], sorted[rightStart:]
}

func positionOfNode(node postingsTreeNode) postingsTreePosition {
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

func emptySubtreeHashesByHeight() [postingsTreeDepth + 1][sha256.Size]byte {
	var hashes [postingsTreeDepth + 1][sha256.Size]byte
	for height := range hashes {
		hashes[height] = sha256.Sum256([]byte{emptySubtreeDomain, byte(height)})
	}

	return hashes
}

func (s postingsSubtree) entriesAround(keptPositions []postingsTreePosition) []postingsTreeNode {
	if len(keptPositions) == 0 {
		return s.summaryEntries()
	}
	if s.position.depth == postingsTreeDepth {
		return nil
	}
	left, right := s.children()
	keptLeft, keptRight := splitAt(s.position.depth, keptPositions, positionItself)

	return append(left.entriesAround(keptLeft), right.entriesAround(keptRight)...)
}

func positionItself(position postingsTreePosition) postingsTreePosition {
	return position
}

func (s postingsSubtree) summaryEntries() []postingsTreeNode {
	hash := s.hash()
	if hash == emptySubtreeHashes[s.position.height()] {
		return nil
	}

	return []postingsTreeNode{{hash: hash, position: s.position}}
}
