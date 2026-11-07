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

func NewSecret() string {
	secret := make([]byte, secretByteLength)
	_, _ = rand.Read(secret)
	return base64.RawURLEncoding.EncodeToString(secret)
}

func SecretIDOf(secret string) string {
	return shortHashOf([]byte(secret))
}

func ChallengeFor(secret string, brokerKeyHash string) []byte {
	challenge := sha256.Sum256([]byte(challengeLabel + secret + brokerKeyHash))
	return challenge[:]
}

func PairingInboxPrefixFor(secret string) string {
	return pairingInboxStart + SecretIDOf(secret)
}

func PairingRepliesFor(secret string) string {
	return PairingInboxPrefixFor(secret) + ".>"
}

func SecretIDFromReply(replySubject string) string {
	secretID, _, found := strings.Cut(strings.TrimPrefix(replySubject, pairingInboxStart), ".")
	if !found || !strings.HasPrefix(replySubject, pairingInboxStart) {
		return ""
	}
	return secretID
}

func shortHashOf(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(hash[:idByteLength])
}
