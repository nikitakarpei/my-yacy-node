package yacymodel

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"
)

var ErrBadPeerKey = errors.New("bad peer key")

type PeerKey struct {
	material keyMaterial
	identity PeerIdentity
}

func ParsePeerKey(text string) (PeerKey, error) {
	parts, separated := partsOf(text)
	if !separated {
		return PeerKey{}, fmt.Errorf("%w: want 3 parts", ErrBadPeerKey)
	}

	var (
		operationalSeed [ed25519.SeedSize]byte
		recoverySeeds   [recoverySeedsSize]byte
	)
	if err := decodeExactly(parts[0], operationalSeed[:]); err != nil {
		return PeerKey{}, fmt.Errorf("%w: operational key: %w", ErrBadPeerKey, err)
	}
	if err := decodeExactly(parts[1], recoverySeeds[:]); err != nil {
		return PeerKey{}, fmt.Errorf("%w: recovery key: %w", ErrBadPeerKey, err)
	}
	nonce, err := nonceOf(parts[2])
	if err != nil {
		return PeerKey{}, fmt.Errorf("%w: nonce: %w", ErrBadPeerKey, err)
	}

	material, err := keyMaterialFrom(operationalSeed, recoverySeeds)
	if err != nil {
		return PeerKey{}, err
	}

	return material.provenBy(nonce)
}

func NewPeerKey() (PeerKey, error) {
	return GeneratePeerKey(rand.Reader)
}

func GeneratePeerKey(entropy io.Reader) (PeerKey, error) {
	var (
		operationalSeed [ed25519.SeedSize]byte
		recoverySeeds   [recoverySeedsSize]byte
	)
	if _, err := io.ReadFull(entropy, operationalSeed[:]); err != nil {
		return PeerKey{}, fmt.Errorf("read entropy: %w", err)
	}
	if _, err := io.ReadFull(entropy, recoverySeeds[:]); err != nil {
		return PeerKey{}, fmt.Errorf("read entropy: %w", err)
	}

	material, err := keyMaterialFrom(operationalSeed, recoverySeeds)
	if err != nil {
		return PeerKey{}, err
	}

	return material.provenBy(material.solvedNonce())
}

func (k PeerKey) Identity() PeerIdentity {
	return k.identity
}

func (k PeerKey) String() string {
	return k.material.String() + identityPartSeparator + nonceTextOf(k.identity.nonce)
}

func (k PeerKey) AddressSignatureOf(
	seed Seed,
	networkName string,
	issuedAt time.Time,
) AddressSignature {
	signature := AddressSignature{IssuedAt: issuedAt.UTC().Truncate(time.Second)}
	copy(signature.Signature[:], ed25519.Sign(
		k.material.operationalKey,
		addressClaimOf(seed, networkName, signature.IssuedAt),
	))

	return signature
}
