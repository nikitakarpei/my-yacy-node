package companionprotocol_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/companionprotocol"
)

func newDeviceKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	return key
}

func TestAProofHoldsForItsKeyAndChallengeOnly(t *testing.T) {
	key := newDeviceKey(t)
	challenge := companionprotocol.ChallengeFrom("secret", "key-hash")

	proof, err := companionprotocol.ProofOf(key, challenge)
	if err != nil {
		t.Fatalf("ProofOf: %v", err)
	}

	if !companionprotocol.IsProofOf(&key.PublicKey, challenge, proof) {
		t.Error("the proof does not hold for its key and challenge")
	}
	other := companionprotocol.ChallengeFrom("other secret", "key-hash")
	if companionprotocol.IsProofOf(&key.PublicKey, other, proof) {
		t.Error("the proof holds for another challenge")
	}
	if companionprotocol.IsProofOf(&newDeviceKey(t).PublicKey, challenge, proof) {
		t.Error("the proof holds for another key")
	}
}

func TestEachKeyGivesItsOwnDeviceIDAndHash(t *testing.T) {
	first, err := x509.MarshalPKIXPublicKey(&newDeviceKey(t).PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}
	second, err := x509.MarshalPKIXPublicKey(&newDeviceKey(t).PublicKey)
	if err != nil {
		t.Fatalf("MarshalPKIXPublicKey: %v", err)
	}

	if companionprotocol.DeviceIDOf(first) == companionprotocol.DeviceIDOf(second) {
		t.Error("two keys give the same device id")
	}
	if companionprotocol.BrokerKeyHashOf(first) == companionprotocol.BrokerKeyHashOf(second) {
		t.Error("two keys give the same hash")
	}
	again := first
	if companionprotocol.DeviceIDOf(first) != companionprotocol.DeviceIDOf(again) {
		t.Error("one key gives two device ids")
	}
}
