package yacymodel

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
)

const postingsProofFormat byte = 1

var ErrBadPostingsProof = errors.New("bad postings proof")

type PostingsProof struct {
	entries           []postingsTreeNode
	partitionExponent uint8
}

func EmptyPostingsProof(partitions DHTRingPartitions) PostingsProof {
	return PostingsProof{partitionExponent: partitions.exponentByte()}
}

func (p PostingsProof) MergedWith(other PostingsProof) PostingsProof {
	if other.partitionExponent != p.partitionExponent {
		return p
	}

	return PostingsProof{
		entries:           entriesUnionOf(p.entries, other.entries),
		partitionExponent: p.partitionExponent,
	}
}

func entriesUnionOf(entries, addedEntries []postingsTreeNode) []postingsTreeNode {
	union := slices.Concat(entries, addedEntries)
	slices.SortStableFunc(union, comparePositions)

	return slices.CompactFunc(union, func(first, second postingsTreeNode) bool {
		return first.position == second.position
	})
}

func (p PostingsProof) NarrowedTo(
	remainingWords []Hash,
	purgedPostings []RWIPosting,
) PostingsProof {
	remainingPositions := wordPositionsOf(remainingWords, p.partitionExponent)
	knownSubtrees := postingsSubtree{entries: outermostNodesOf(
		entriesBesides(p.entries, remainingPositions),
		leavesOf(purgedPostings, p.partitionExponent),
	)}

	return PostingsProof{
		entries:           knownSubtrees.entriesAround(remainingPositions),
		partitionExponent: p.partitionExponent,
	}
}

func wordPositionsOf(words []Hash, partitionExponent uint8) []postingsTreePosition {
	positions := make([]postingsTreePosition, 0, len(words))
	for _, word := range words {
		positions = append(positions, leafPositionOf(word, partitionExponent))
	}
	slices.SortFunc(positions, postingsTreePosition.compare)

	return positions
}

func entriesBesides(
	entries []postingsTreeNode,
	sortedPositions []postingsTreePosition,
) []postingsTreeNode {
	return slices.DeleteFunc(slices.Clone(entries), func(entry postingsTreeNode) bool {
		next, _ := slices.BinarySearchFunc(
			sortedPositions,
			entry.position,
			postingsTreePosition.compare,
		)

		return next < len(sortedPositions) && entry.position.covers(sortedPositions[next])
	})
}

func outermostNodesOf(entries, leaves []postingsTreeNode) []postingsTreeNode {
	var outermostNodes []postingsTreeNode
	for _, node := range entriesUnionOf(entries, leaves) {
		if len(outermostNodes) == 0 ||
			!outermostNodes[len(outermostNodes)-1].position.covers(node.position) {
			outermostNodes = append(outermostNodes, node)
		}
	}

	return outermostNodes
}

func (p PostingsProof) liesBesidePathsOf(postings []RWIPosting) bool {
	leafPositions := leafPositionsOf(postings, p.partitionExponent)

	return !slices.ContainsFunc(p.entries, func(entry postingsTreeNode) bool {
		pathNode, isBesidePath := entry.position.sibling()
		next, _ := slices.BinarySearchFunc(
			leafPositions,
			pathNode,
			postingsTreePosition.compare,
		)

		return !isBesidePath || next == len(leafPositions) ||
			!pathNode.covers(leafPositions[next])
	})
}

func (p PostingsProof) Bytes() []byte {
	encoding := []byte{postingsProofFormat, p.partitionExponent}
	for _, entry := range p.entries {
		encoding = entry.position.appendedTo(encoding)
		encoding = append(encoding, entry.hash[:]...)
	}

	return encoding
}

func ParsePostingsProofBytes(raw []byte) (PostingsProof, error) {
	if len(raw) < 2 || raw[0] != postingsProofFormat {
		return PostingsProof{}, fmt.Errorf(
			"%w: no format %d header",
			ErrBadPostingsProof,
			postingsProofFormat,
		)
	}
	if _, err := DHTRingPartitionsFromExponent(uint(raw[1])); err != nil {
		return PostingsProof{}, fmt.Errorf("%w: %w", ErrBadPostingsProof, err)
	}
	entries, err := entriesFrom(raw[2:])
	if err != nil {
		return PostingsProof{}, err
	}

	return PostingsProof{entries: entries, partitionExponent: raw[1]}, nil
}

func entriesFrom(raw []byte) ([]postingsTreeNode, error) {
	var entries []postingsTreeNode
	for len(raw) > 0 {
		entry, rest, err := entryFrom(raw)
		if err != nil {
			return nil, err
		}
		if len(entries) > 0 && entries[len(entries)-1].position.compare(entry.position) >= 0 {
			return nil, fmt.Errorf("%w: entries out of order", ErrBadPostingsProof)
		}
		entries, raw = append(entries, entry), rest
	}

	return entries, nil
}

func entryFrom(raw []byte) (entry postingsTreeNode, rest []byte, err error) {
	entry.position, rest, err = positionFrom(raw)
	if err != nil {
		return entry, nil, fmt.Errorf("%w: %w", ErrBadPostingsProof, err)
	}
	if len(rest) < sha256.Size {
		return entry, nil, fmt.Errorf("%w: truncated hash", ErrBadPostingsProof)
	}
	copy(entry.hash[:], rest)

	return entry, rest[sha256.Size:], nil
}
