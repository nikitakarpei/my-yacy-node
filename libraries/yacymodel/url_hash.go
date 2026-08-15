package yacymodel

import (
	"fmt"
)

const hostHashLength = 6

type URLHash struct{ hash Hash }

func ParseURLHash(raw string) (URLHash, error) {
	hash, err := ParseHash(raw)
	if err != nil {
		return URLHash{}, fmt.Errorf("parse url hash: %w", err)
	}

	return URLHash{hash: hash}, nil
}

func (h URLHash) MarshalText() ([]byte, error) {
	return h.hash.MarshalText()
}

func (h *URLHash) UnmarshalText(text []byte) error {
	parsed, err := ParseURLHash(string(text))
	if err != nil {
		return err
	}
	*h = parsed

	return nil
}

func (h URLHash) IsZero() bool {
	return h.hash.IsZero()
}

func (h URLHash) String() string {
	return h.hash.value
}

func (h URLHash) HostHash() HostHash {
	return HostHash{value: h.hash.value[HashLength-hostHashLength:]}
}
