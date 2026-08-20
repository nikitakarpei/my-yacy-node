package vaultenginetest

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var stringKeyLayout = vault.SingleKey(vault.TextKeyPart)

type stringKeyCodec struct{}

func (stringKeyCodec) Encode(key string) vault.Key { return stringKeyLayout.Key(key) }

func (stringKeyCodec) Decode(storedKey []byte) (string, error) {
	decoded, err := stringKeyLayout.Parts(storedKey)
	if err != nil {
		return "", fmt.Errorf("conformance key: %w", err)
	}

	return decoded, nil
}

type stringValueCodec struct{}

func (stringValueCodec) Encode(value string) ([]byte, error) { return []byte(value), nil }
func (stringValueCodec) Decode(raw []byte) (string, error)   { return string(raw), nil }
