package yacymodel

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"

	"github.com/cloudflare/circl/sign/slhdsa"
)

const recoveryScheme = slhdsa.SHA2_128s

const recoverySeedsSize = 48

type peerSecret struct {
	operationalKey ed25519.PrivateKey
	recoverySeeds  [recoverySeedsSize]byte
}

func peerSecretFrom(
	operationalSeed [ed25519.SeedSize]byte,
	recoverySeeds [recoverySeedsSize]byte,
) peerSecret {
	return peerSecret{
		operationalKey: ed25519.NewKeyFromSeed(operationalSeed[:]),
		recoverySeeds:  recoverySeeds,
	}
}

func (s peerSecret) recoveryCommitment() ([sha256.Size]byte, error) {
	recoveryPublicKey, _, err := slhdsa.GenerateKey(
		bytes.NewReader(s.recoverySeeds[:]),
		recoveryScheme,
	)
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("%w: recovery key: %w", ErrBadPeerKey, err)
	}
	encodedKey, err := recoveryPublicKey.MarshalBinary()
	if err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("%w: recovery key: %w", ErrBadPeerKey, err)
	}

	return sha256.Sum256(encodedKey), nil
}

func (s peerSecret) operationalPublicKey() [ed25519.PublicKeySize]byte {
	return [ed25519.PublicKeySize]byte(s.operationalKey.Public().(ed25519.PublicKey))
}

func (s peerSecret) signatureFrom(claim []byte) [ed25519.SignatureSize]byte {
	return [ed25519.SignatureSize]byte(ed25519.Sign(s.operationalKey, claim))
}

func (s peerSecret) text() string {
	return Encode(s.operationalKey.Seed()) + identityPartSeparator + Encode(s.recoverySeeds[:])
}
