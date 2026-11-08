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

func SecretIDOf(secret Secret) SecretID {
	return SecretID(shortHashOf([]byte(secret)))
}

func shortHashOf(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(hash[:idByteLength])
}
