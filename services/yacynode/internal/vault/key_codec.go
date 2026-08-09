package vault

import "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"

type KeyCodec[K any] interface {
	Encode(K) vaultkey.Key
	Decode([]byte) (K, error)
}
