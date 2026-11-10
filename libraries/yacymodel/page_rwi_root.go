package yacymodel

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	pageRWIRootDomain        byte = 1
	attestedMetadataFormat   byte = 1
	pageRWIRootByteLength         = sha256.Size
	absentOptionalFieldMark  byte = 0
	presentOptionalFieldMark byte = 1
)

var ErrBadPageRWIRoot = errors.New("bad page RWI root")

type PageRWIRoot struct {
	digest [pageRWIRootByteLength]byte
}

func PageRWIRootFrom(metadata URLMetadata) Optional[PageRWIRoot] {
	postingsRoot, rooted := metadata.PostingsRoot.Get()
	if !rooted {
		return None[PageRWIRoot]()
	}
	metadataDigest := sha256.Sum256(attestedMetadataOf(metadata))
	committed := append([]byte{pageRWIRootDomain}, metadataDigest[:]...)

	return Some(PageRWIRoot{digest: sha256.Sum256(append(committed, postingsRoot.Bytes()...))})
}

func attestedMetadataOf(metadata URLMetadata) []byte {
	encoding := []byte{attestedMetadataFormat}
	encoding = metadata.Hash.AppendBytes(encoding)
	encoding = appendOptional(encoding, metadata.Referrer, appendURLHash)
	encoding = append(encoding, byte(metadata.DocumentType))
	encoding = appendOptional(encoding, metadata.Language, appendLanguage)
	encoding = appendOptional(encoding, metadata.ByteSize, binary.BigEndian.AppendUint32)
	for _, count := range []uint32{
		metadata.WordCount,
		metadata.LocalLinks,
		metadata.ExternalLinks,
		metadata.ImageLinks,
		metadata.AudioLinks,
		metadata.VideoLinks,
		metadata.ApplicationLinks,
	} {
		encoding = binary.BigEndian.AppendUint32(encoding, count)
	}
	encoding = appendText(encoding, metadata.Address)
	encoding = appendText(encoding, metadata.Title)
	encoding = appendText(encoding, metadata.Author)
	encoding = appendTexts(encoding, metadata.Tags)
	encoding = appendText(encoding, metadata.Publisher)
	encoding = appendOptional(encoding, metadata.Location, appendCoordinates)
	encoding = appendOptional(encoding, metadata.Modified, appendCalendarDay)
	encoding = appendOptional(encoding, metadata.FreshUntil, appendCalendarDay)
	encoding = appendText(encoding, metadata.MediaType)
	encoding = appendText(encoding, metadata.Snippet)

	return appendText(encoding, metadata.FaviconAddress)
}

func appendOptional[T any](
	encoding []byte,
	field Optional[T],
	appendValue func([]byte, T) []byte,
) []byte {
	value, present := field.Get()
	if !present {
		return append(encoding, absentOptionalFieldMark)
	}

	return appendValue(append(encoding, presentOptionalFieldMark), value)
}

func appendURLHash(encoding []byte, hash URLHash) []byte {
	return hash.AppendBytes(encoding)
}

func appendLanguage(encoding []byte, language Language) []byte {
	return language.AppendBytes(encoding)
}

func appendText(encoding []byte, text string) []byte {
	return append(binary.AppendUvarint(encoding, uint64(len(text))), text...)
}

func appendTexts(encoding []byte, texts []string) []byte {
	encoding = binary.AppendUvarint(encoding, uint64(len(texts)))
	for _, text := range texts {
		encoding = appendText(encoding, text)
	}

	return encoding
}

func appendCoordinates(encoding []byte, location Coordinates) []byte {
	encoding = binary.BigEndian.AppendUint64(encoding, math.Float64bits(location.Latitude))

	return binary.BigEndian.AppendUint64(encoding, math.Float64bits(location.Longitude))
}

func appendCalendarDay(encoding []byte, day CalendarDay) []byte {
	return binary.AppendVarint(encoding, day.Time().Unix())
}

func (r PageRWIRoot) Bytes() []byte {
	return r.digest[:]
}

func ParsePageRWIRootBytes(raw []byte) (PageRWIRoot, error) {
	if len(raw) != pageRWIRootByteLength {
		return PageRWIRoot{}, fmt.Errorf(
			"%w: %d bytes, want %d", ErrBadPageRWIRoot, len(raw), pageRWIRootByteLength,
		)
	}
	var root PageRWIRoot
	copy(root.digest[:], raw)

	return root, nil
}
