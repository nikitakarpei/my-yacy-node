package yacymodel

import (
	"crypto/rand"
	"fmt"
	"io"
)

func GenerateHash(entropy io.Reader) (Hash, error) {
	hashBytes := make([]byte, HashByteLength)
	if _, err := io.ReadFull(entropy, hashBytes); err != nil {
		return Hash{}, fmt.Errorf("read entropy: %w", err)
	}

	return ParseHashBytes(hashBytes)
}

func NewHash() (Hash, error) {
	return GenerateHash(rand.Reader)
}
