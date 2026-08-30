package vaultenginetest

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
)

var stringKeyParts = vault.SingleKey(vault.TextKeyPart)

var stringKeyLayout = stringKeyParts.KeyLayout()

type stringValueCodec struct{}

func (stringValueCodec) Encode(value string) ([]byte, error) { return []byte(value), nil }
func (stringValueCodec) Decode(raw []byte) (string, error)   { return string(raw), nil }
