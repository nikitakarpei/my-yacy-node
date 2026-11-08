package companionprotocol

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

const (
	secretByteLength  = 16
	idByteLength      = 16
	challengeLabel    = "yacypeer pairing 1"
	pairingInboxStart = "pairing.replies."

	PairingRequestSubject = "pairing.request"
)

type Secret string

type SecretID string

func NewSecret() Secret {
	secret := make([]byte, secretByteLength)
	_, _ = rand.Read(secret)
	return Secret(base64.RawURLEncoding.EncodeToString(secret))
}

func SecretIDOf(secret Secret) SecretID {
	return SecretID(shortHashOf([]byte(secret)))
}

func ChallengeFor(secret Secret, brokerKeyHash string) []byte {
	challenge := sha256.Sum256([]byte(challengeLabel + string(secret) + brokerKeyHash))
	return challenge[:]
}

func PairingInboxPrefixFor(secret Secret) string {
	return pairingInboxStart + string(SecretIDOf(secret))
}

func PairingRepliesFor(secret Secret) string {
	return PairingInboxPrefixFor(secret) + ".>"
}

func SecretIDFromReply(replySubject string) SecretID {
	secretID, _, found := strings.Cut(strings.TrimPrefix(replySubject, pairingInboxStart), ".")
	if !found || !strings.HasPrefix(replySubject, pairingInboxStart) {
		return ""
	}
	return SecretID(secretID)
}

func shortHashOf(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(hash[:idByteLength])
}
