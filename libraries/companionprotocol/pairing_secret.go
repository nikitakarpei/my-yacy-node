package companionprotocol

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const idByteLength = 16

type Secret string

type SecretID string

func RandomSecret() Secret {
	return Secret(rand.Text())
}

func SecretIDFrom(secret Secret) SecretID {
	return SecretID(shortHashFrom([]byte(secret)))
}

func shortHashFrom(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(hash[:idByteLength])
}
