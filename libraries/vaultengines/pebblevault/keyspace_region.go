package pebblevault

import (
	"encoding/binary"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	regionPrefixLength       = 1
	recordsRegionPrefix byte = 1
	tallyRegionPrefix   byte = 2
	amountsRegionPrefix byte = 3
)

type keyspaceRegion struct {
	prefix []byte
}

var amountsRegion = keyspaceRegion{prefix: []byte{amountsRegionPrefix}}

func recordsRegionOf(bucket vault.Name) keyspaceRegion {
	return namedRegionWithin(recordsRegionPrefix, bucket)
}

func namedRegionWithin(regionPrefix byte, bucket vault.Name) keyspaceRegion {
	prefix := binary.AppendUvarint([]byte{regionPrefix}, uint64(len(bucket)))

	return keyspaceRegion{prefix: append(prefix, bucket...)}
}

func bucketNameFrom(namedRegionKey []byte) vault.Name {
	nameLength, lengthBytes := binary.Uvarint(namedRegionKey[regionPrefixLength:])
	nameStart := regionPrefixLength + lengthBytes
	//nolint:gosec // G115: a bucket name is far shorter than MaxInt.
	nameEnd := nameStart + int(nameLength)

	return vault.Name(namedRegionKey[nameStart:nameEnd])
}

func amountsRegionOf(bucket vault.Name) keyspaceRegion {
	return namedRegionWithin(amountsRegionPrefix, bucket)
}

func (r keyspaceRegion) absoluteKeyFrom(relativeKey []byte) []byte {
	absolute := make([]byte, 0, len(r.prefix)+len(relativeKey))
	absolute = append(absolute, r.prefix...)

	return append(absolute, relativeKey...)
}

func (r keyspaceRegion) relativeKeyFrom(absoluteKey []byte) []byte {
	return absoluteKey[len(r.prefix):]
}

func (r keyspaceRegion) boundsFor(keys vault.KeyRange) (firstIncluded, firstExcluded []byte) {
	included, excluded := keys.Bounds()
	if excluded == nil {
		return r.absoluteKeyFrom(included), r.firstAbsoluteKeyAfter()
	}

	return r.absoluteKeyFrom(included), r.absoluteKeyFrom(excluded)
}

func (r keyspaceRegion) firstAbsoluteKeyAfter() []byte {
	after := r.absoluteKeyFrom(nil)
	for after[len(after)-1] == 0xff {
		after = after[:len(after)-1]
	}
	after[len(after)-1]++

	return after
}
