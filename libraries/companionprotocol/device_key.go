package companionprotocol

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func BrokerKeyHashOf(publicKeyInfo []byte) string {
	hash := sha256.Sum256(publicKeyInfo)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func DeviceIDOf(publicKeyInfo []byte) string {
	return shortHashOf(publicKeyInfo)
}

func ProofOf(key crypto.Signer, challenge []byte) ([]byte, error) {
	digest := sha256.Sum256(challenge)
	proof, err := key.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("sign the pairing challenge: %w", err)
	}
	return proof, nil
}

func IsProofOf(publicKey *ecdsa.PublicKey, challenge []byte, proof []byte) bool {
	digest := sha256.Sum256(challenge)
	return ecdsa.VerifyASN1(publicKey, digest[:], proof)
}
