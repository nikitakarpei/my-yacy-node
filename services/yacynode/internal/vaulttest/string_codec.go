package vaulttest

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

var stringKeyLayout = vaultkey.Single(vaultkey.Text)

type stringKeyCodec struct{}

func (stringKeyCodec) Encode(key string) vaultkey.Key { return stringKeyLayout.Key(key) }

func (stringKeyCodec) Decode(key vaultkey.Key) (string, error) {
	decoded, err := stringKeyLayout.Parts(key)
	if err != nil {
		return "", fmt.Errorf("conformance key: %w", err)
	}

	return decoded, nil
}

type stringValueCodec struct{}

func (stringValueCodec) Encode(value string) ([]byte, error) { return []byte(value), nil }
func (stringValueCodec) Decode(raw []byte) (string, error)   { return string(raw), nil }
