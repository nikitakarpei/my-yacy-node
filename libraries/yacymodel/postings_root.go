package yacymodel

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

const (
	postingsRootFormat     byte = 1
	postingsRootByteLength      = 2 + sha256.Size
)

var ErrBadPostingsRoot = errors.New("bad postings root")

type PostingsRoot struct {
	digest            [sha256.Size]byte
	partitionExponent uint8
}

func (r PostingsRoot) Bytes() []byte {
	return append([]byte{postingsRootFormat, r.partitionExponent}, r.digest[:]...)
}

func ParsePostingsRootBytes(raw []byte) (PostingsRoot, error) {
	if len(raw) != postingsRootByteLength || raw[0] != postingsRootFormat {
		return PostingsRoot{}, fmt.Errorf(
			"%w: %d bytes, want %d in format %d",
			ErrBadPostingsRoot,
			len(raw),
			postingsRootByteLength,
			postingsRootFormat,
		)
	}
	if _, err := DHTRingPartitionsFromExponent(uint(raw[1])); err != nil {
		return PostingsRoot{}, fmt.Errorf("%w: %w", ErrBadPostingsRoot, err)
	}
	root := PostingsRoot{partitionExponent: raw[1]}
	copy(root.digest[:], raw[2:])

	return root, nil
}
