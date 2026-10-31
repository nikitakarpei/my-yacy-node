package yacymodel

import (
	"crypto/sha256"
	"encoding/binary"
	"slices"
)

const (
	leafDomain byte = iota
	innerNodeDomain
	emptySubtreeDomain
)

type postingsTreeNode struct {
	hash     [sha256.Size]byte
	position postingsTreePosition
}

func leavesOf(postings []RWIPosting, partitionExponent uint8) []postingsTreeNode {
	leaves := make([]postingsTreeNode, 0, len(postings))
	for _, posting := range postings {
		leaves = append(leaves, leafOf(posting, partitionExponent))
	}
	slices.SortFunc(leaves, comparePositions)

	return leaves
}

func leafOf(posting RWIPosting, partitionExponent uint8) postingsTreeNode {
	return postingsTreeNode{
		hash:     sha256.Sum256(leafEncodingOf(posting)),
		position: leafPositionOf(posting.WordHash, partitionExponent),
	}
}

func leafEncodingOf(posting RWIPosting) []byte {
	encoding := []byte{leafDomain}
	encoding = append(encoding, posting.WordHash.raw[:]...)
	encoding = append(encoding, posting.URLHash.hash.raw[:]...)
	encoding = binary.BigEndian.AppendUint16(encoding, uint16(posting.LastModified))
	encoding = append(encoding, posting.Language.code[:]...)
	encoding = append(encoding, byte(posting.DocumentType), posting.TitleWords)
	encoding = binary.BigEndian.AppendUint16(encoding, posting.TextWords)
	encoding = binary.BigEndian.AppendUint16(encoding, posting.Phrases)
	encoding = append(
		encoding,
		posting.LocalLinks,
		posting.ExternalLinks,
		posting.URLLength,
		posting.URLComponents,
	)
	encoding = binary.BigEndian.AppendUint16(encoding, appearanceBitsOf(posting.Appearance))
	encoding = binary.BigEndian.AppendUint16(encoding, posting.TextPosition)

	return append(
		encoding,
		posting.Hits,
		posting.PhraseRelativePosition,
		posting.PhrasePosition,
	)
}

func appearanceBitsOf(appearance Appearance) uint16 {
	var appearanceBits uint16
	for i, trait := range []bool{
		appearance.IndexOf,
		appearance.HasLocation,
		appearance.HasImage,
		appearance.HasAudio,
		appearance.HasVideo,
		appearance.HasApp,
		appearance.AppearsInDescription,
		appearance.AppearsInTitle,
		appearance.AppearsInCreator,
		appearance.AppearsInSubject,
		appearance.AppearsInIdentifier,
		appearance.Emphasized,
	} {
		if trait {
			appearanceBits |= 1 << i
		}
	}

	return appearanceBits
}

func leafPositionOf(word Hash, partitionExponent uint8) postingsTreePosition {
	var position postingsTreePosition
	for position.depth < postingsTreeDepth {
		index := (uint(position.depth) + uint(partitionExponent)) % postingsTreeDepth
		position = position.childAt(bitOf(word.raw[:], index))
	}

	return position
}

func comparePositions(first, second postingsTreeNode) int {
	return first.position.compare(second.position)
}

func leafPositionsOf(postings []RWIPosting, partitionExponent uint8) []postingsTreePosition {
	positions := make([]postingsTreePosition, 0, len(postings))
	for _, posting := range postings {
		positions = append(positions, leafPositionOf(posting.WordHash, partitionExponent))
	}
	slices.SortFunc(positions, postingsTreePosition.compare)

	return positions
}
