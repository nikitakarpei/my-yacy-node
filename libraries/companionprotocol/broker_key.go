package companionprotocol

import (
	"crypto/sha256"
	"encoding/base64"
)

type BrokerKeyHash string

func BrokerKeyHashOf(publicKeyInfo []byte) BrokerKeyHash {
	hash := sha256.Sum256(publicKeyInfo)
	return BrokerKeyHash(base64.RawURLEncoding.EncodeToString(hash[:]))
}
