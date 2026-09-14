package yacymodel

import (
	"fmt"
	"net/url"
)

type URLHash struct{ hash Hash }

func URLHashOf(address string) (URLHash, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return URLHash{}, fmt.Errorf("url hash: address %q: %w", address, err)
	}

	return URLNormalformOf(parsed).Hash(), nil
}

func ParseURLHash(raw string) (URLHash, error) {
	hash, err := ParseHash(raw)
	if err != nil {
		return URLHash{}, fmt.Errorf("parse url hash: %w", err)
	}

	return URLHash{hash: hash}, nil
}

func ParseURLHashBytes(raw []byte) (URLHash, error) {
	hash, err := ParseHashBytes(raw)
	if err != nil {
		return URLHash{}, fmt.Errorf("parse url hash: %w", err)
	}

	return URLHash{hash: hash}, nil
}

func (h URLHash) Bytes() []byte {
	return h.hash.Bytes()
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
	return HostHash{value: h.hash.value[HashLength-HostHashLength:]}
}
