package featurehashing

import (
	"encoding/binary"
	"math/bits"
)

const (
	murmur3BlockBytes   = 4
	murmur3FirstFactor  = 0xcc9e2d51
	murmur3SecondFactor = 0x1b873593
)

func murmur3Of(data []byte) int32 {
	var hash uint32
	blockAmount := len(data) / murmur3BlockBytes
	for block := range blockAmount {
		hash ^= murmur3Mixed(binary.LittleEndian.Uint32(data[block*murmur3BlockBytes:]))
		hash = bits.RotateLeft32(hash, 13)*5 + 0xe6546b64
	}
	var tail uint32
	for position, tailByte := range data[blockAmount*murmur3BlockBytes:] {
		tail |= uint32(tailByte) << (8 * position)
	}
	if len(data)%murmur3BlockBytes != 0 {
		hash ^= murmur3Mixed(tail)
	}
	lengthBits := uint32(len(data))                   //nolint:gosec // length modulo 2^32
	return int32(murmur3Finalized(hash ^ lengthBits)) //nolint:gosec // signed like scikit-learn
}

func murmur3Mixed(block uint32) uint32 {
	return bits.RotateLeft32(block*murmur3FirstFactor, 15) * murmur3SecondFactor
}

func murmur3Finalized(hash uint32) uint32 {
	hash ^= hash >> 16
	hash *= 0x85ebca6b
	hash ^= hash >> 13
	hash *= 0xc2b2ae35
	return hash ^ hash>>16
}
