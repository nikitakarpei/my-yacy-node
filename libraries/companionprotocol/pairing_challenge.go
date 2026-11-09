package companionprotocol

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
)

const challengeLabel = "yacypeer pairing 1"

type Challenge []byte

type Proof []byte

func ChallengeFrom(secret Secret, brokerKeyHash BrokerKeyHash) Challenge {
	challenge := sha256.Sum256([]byte(challengeLabel + string(secret) + string(brokerKeyHash)))
	return challenge[:]
}

func ProofOf(signer crypto.Signer, challenge Challenge) (Proof, error) {
	digest := sha256.Sum256(challenge)
	proof, err := signer.Sign(rand.Reader, digest[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("sign the pairing challenge: %w", err)
	}
	return proof, nil
}

func (p Proof) HoldsFor(publicKey *ecdsa.PublicKey, challenge Challenge) bool {
	digest := sha256.Sum256(challenge)
	return ecdsa.VerifyASN1(publicKey, digest[:], p)
}
