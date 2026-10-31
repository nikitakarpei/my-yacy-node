package postingstree

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
)

var errBadEntries = errors.New("bad postings tree entries")

func UnionOf(entries, addedEntries []Node) []Node {
	union := slices.Concat(entries, addedEntries)
	slices.SortStableFunc(union, compareNodes)

	return slices.CompactFunc(union, func(first, second Node) bool {
		return first.position == second.position
	})
}

func EntriesBesides(entries []Node, positions []Position) []Node {
	sortedPositions := sortedPositionsOf(positions)

	return slices.DeleteFunc(slices.Clone(entries), func(entry Node) bool {
		next, _ := slices.BinarySearchFunc(sortedPositions, entry.position, Position.compare)

		return next < len(sortedPositions) && entry.position.covers(sortedPositions[next])
	})
}

func sortedPositionsOf(positions []Position) []Position {
	sorted := slices.Clone(positions)
	slices.SortFunc(sorted, Position.compare)

	return sorted
}

func OutermostNodesOf(entries, leaves []Node) []Node {
	var outermostNodes []Node
	for _, node := range UnionOf(entries, leaves) {
		if len(outermostNodes) == 0 ||
			!outermostNodes[len(outermostNodes)-1].position.covers(node.position) {
			outermostNodes = append(outermostNodes, node)
		}
	}

	return outermostNodes
}

func LieBesidePathsOf(entries []Node, leafPositions []Position) bool {
	sortedLeafPositions := sortedPositionsOf(leafPositions)

	return !slices.ContainsFunc(entries, func(entry Node) bool {
		pathNode, isBesidePath := entry.position.sibling()
		next, _ := slices.BinarySearchFunc(sortedLeafPositions, pathNode, Position.compare)

		return !isBesidePath || next == len(sortedLeafPositions) ||
			!pathNode.covers(sortedLeafPositions[next])
	})
}

func AppendEntries(encoding []byte, entries []Node) []byte {
	for _, entry := range entries {
		encoding = entry.position.appendedTo(encoding)
		encoding = append(encoding, entry.hash[:]...)
	}

	return encoding
}

func EntriesFrom(raw []byte) ([]Node, error) {
	var entries []Node
	for len(raw) > 0 {
		entry, rest, err := entryFrom(raw)
		if err != nil {
			return nil, err
		}
		if len(entries) > 0 && entries[len(entries)-1].position.compare(entry.position) >= 0 {
			return nil, fmt.Errorf("%w: entries out of order", errBadEntries)
		}
		entries, raw = append(entries, entry), rest
	}

	return entries, nil
}

func entryFrom(raw []byte) (entry Node, rest []byte, err error) {
	entry.position, rest, err = positionFrom(raw)
	if err != nil {
		return entry, nil, err
	}
	if len(rest) < sha256.Size {
		return entry, nil, fmt.Errorf("%w: truncated hash", errBadEntries)
	}
	copy(entry.hash[:], rest)

	return entry, rest[sha256.Size:], nil
}
