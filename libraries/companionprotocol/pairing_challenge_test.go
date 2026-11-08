package companionprotocol_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func TestTheChallengeDependsOnTheSecretAndTheBrokerKey(t *testing.T) {
	challenge := companionprotocol.ChallengeFrom("secret-a", "key-a")

	if !bytes.Equal(challenge, companionprotocol.ChallengeFrom("secret-a", "key-a")) {
		t.Error("the same secret and key give another challenge")
	}
	if bytes.Equal(challenge, companionprotocol.ChallengeFrom("secret-b", "key-a")) {
		t.Error("another secret gives the same challenge")
	}
	if bytes.Equal(challenge, companionprotocol.ChallengeFrom("secret-a", "key-b")) {
		t.Error("another broker key gives the same challenge")
	}
}

func TestAProofHoldsForItsKeyAndChallengeOnly(t *testing.T) {
	key := randomDeviceKey(t)
	challenge := companionprotocol.ChallengeFrom("secret", "key-hash")

	proof, err := companionprotocol.ProofOf(key, challenge)
	if err != nil {
		t.Fatalf("ProofOf: %v", err)
	}

	if !proof.HoldsFor(&key.PublicKey, challenge) {
		t.Error("the proof does not hold for its key and challenge")
	}
	otherChallenge := companionprotocol.ChallengeFrom("other secret", "key-hash")
	if proof.HoldsFor(&key.PublicKey, otherChallenge) {
		t.Error("the proof holds for another challenge")
	}
	if proof.HoldsFor(&randomDeviceKey(t).PublicKey, challenge) {
		t.Error("the proof holds for another key")
	}
}

func randomDeviceKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}
