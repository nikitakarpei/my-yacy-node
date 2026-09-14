package pebblevault

import (
	"encoding/binary"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

const (
	bucketRegionPrefix byte = 1
	tallyRegionPrefix  byte = 2
)

type keyspaceRegion struct {
	prefix []byte
}

func bucketRegionOf(bucket vault.Name) keyspaceRegion {
	prefix := binary.AppendUvarint([]byte{bucketRegionPrefix}, uint64(len(bucket)))

	return keyspaceRegion{prefix: append(prefix, bucket...)}
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
