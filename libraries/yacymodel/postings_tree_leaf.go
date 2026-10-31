package yacymodel

import (
	"encoding/binary"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/internal/postingstree"
)

func leavesOf(postings []RWIPosting, partitionExponent uint8) []postingstree.Node {
	leaves := make([]postingstree.Node, 0, len(postings))
	for _, posting := range postings {
		leaves = append(leaves, postingstree.LeafOf(
			leafEncodingOf(posting),
			leafPositionOf(posting.WordHash, partitionExponent),
		))
	}

	return leaves
}

func leafEncodingOf(posting RWIPosting) []byte {
	encoding := posting.WordHash.AppendBytes(nil)
	encoding = posting.URLHash.AppendBytes(encoding)
	encoding = binary.BigEndian.AppendUint16(encoding, uint16(posting.LastModified))
	encoding = posting.Language.AppendBytes(encoding)
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

func leafPositionsOf(postings []RWIPosting, partitionExponent uint8) []postingstree.Position {
	positions := make([]postingstree.Position, 0, len(postings))
	for _, posting := range postings {
		positions = append(positions, leafPositionOf(posting.WordHash, partitionExponent))
	}

	return positions
}

func wordPositionsOf(words []Hash, partitionExponent uint8) []postingstree.Position {
	positions := make([]postingstree.Position, 0, len(words))
	for _, word := range words {
		positions = append(positions, leafPositionOf(word, partitionExponent))
	}

	return positions
}

func leafPositionOf(word Hash, partitionExponent uint8) postingstree.Position {
	const wordBits = HashByteLength * 8
	var key [HashByteLength]byte
	for bit := range uint(wordBits) {
		wordBit := (bit + uint(partitionExponent)) % wordBits
		key[bit/8] |= (word.raw[wordBit/8] >> (7 - wordBit%8) & 1) << (7 - bit%8)
	}

	return postingstree.LeafAt(key)
}
