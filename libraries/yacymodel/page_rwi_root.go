package yacymodel

import (
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
)

const (
	pageRWIRootDomain      byte = 1
	attestedMetadataFormat byte = 1
	pageRWIRootByteLength       = sha256.Size
)

var ErrBadPageRWIRoot = errors.New("bad page RWI root")

type PageRWIRoot struct {
	digest [pageRWIRootByteLength]byte
}

func PageRWIRootFrom(metadata URLMetadata) Optional[PageRWIRoot] {
	if !metadata.PostingsRoot.Present() {
		return None[PageRWIRoot]()
	}

	return Some(PageRWIRoot{digest: sha256.Sum256(attestedMetadataOf(metadata))})
}

func attestedMetadataOf(metadata URLMetadata) []byte {
	var fields storedfields.Writer
	fields.Byte(pageRWIRootDomain)
	fields.Byte(attestedMetadataFormat)
	fields.Fixed(metadata.Hash.AppendBytes(nil))
	writeOptional(
		&fields,
		metadata.Referrer,
		func(hash URLHash) { fields.Fixed(hash.AppendBytes(nil)) },
	)
	fields.Byte(byte(metadata.DocumentType))
	writeOptional(&fields, metadata.Language, func(language Language) {
		fields.Fixed(language.AppendBytes(nil))
	})
	writeOptional(
		&fields,
		metadata.PostingsRoot,
		func(root PostingsRoot) { fields.Fixed(root.Bytes()) },
	)
	writeOptional(&fields, metadata.ByteSize, func(size uint32) { fields.Count(int(size)) })
	for _, count := range []uint32{
		metadata.WordCount,
		metadata.LocalLinks,
		metadata.ExternalLinks,
		metadata.ImageLinks,
		metadata.AudioLinks,
		metadata.VideoLinks,
		metadata.ApplicationLinks,
	} {
		fields.Count(int(count))
	}
	fields.Text(metadata.Address)
	fields.Text(metadata.Title)
	fields.Text(metadata.Author)
	fields.Count(len(metadata.Tags))
	for _, tag := range metadata.Tags {
		fields.Text(tag)
	}
	fields.Text(metadata.Publisher)
	writeOptional(&fields, metadata.Location, func(location Coordinates) {
		fields.Float(location.Latitude)
		fields.Float(location.Longitude)
	})
	writeOptional(&fields, metadata.Modified, func(day CalendarDay) { fields.Time(day.Time()) })
	// Loaded is left out: a crawl of an unchanged page gives the same root, and the statement holds the fetch time.
	writeOptional(&fields, metadata.FreshUntil, func(day CalendarDay) { fields.Time(day.Time()) })
	fields.Text(metadata.MediaType)
	fields.Text(metadata.Snippet)
	fields.Text(metadata.FaviconAddress)

	return fields.Record()
}

func writeOptional[T any](fields *storedfields.Writer, field Optional[T], writeValue func(T)) {
	value, present := field.Get()
	fields.Presence(present)
	if present {
		writeValue(value)
	}
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
