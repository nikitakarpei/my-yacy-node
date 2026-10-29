package yacymodel

import (
	"context"
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
	var (
		operationalSeed [ed25519.SeedSize]byte
		recoverySeeds   [recoverySeedsSize]byte
	)
	nonce, err := decodeParts(text, operationalSeed[:], recoverySeeds[:])
	if err != nil {
		return PeerKey{}, fmt.Errorf("%w: %w", ErrBadPeerKey, err)
	}

	material, err := keyMaterialFrom(operationalSeed, recoverySeeds)
	if err != nil {
		return PeerKey{}, err
	}

	return material.provenBy(nonce)
}

func NewPeerKey(ctx context.Context) (PeerKey, error) {
	return GeneratePeerKey(ctx, rand.Reader)
}

func GeneratePeerKey(ctx context.Context, entropy io.Reader) (PeerKey, error) {
	var seeds [ed25519.SeedSize + recoverySeedsSize]byte
	if _, err := io.ReadFull(entropy, seeds[:]); err != nil {
		return PeerKey{}, fmt.Errorf("read entropy: %w", err)
	}

	material, err := keyMaterialFrom(
		[ed25519.SeedSize]byte(seeds[:ed25519.SeedSize]),
		[recoverySeedsSize]byte(seeds[ed25519.SeedSize:]),
	)
	if err != nil {
		return PeerKey{}, err
	}

	nonce, err := material.solvedNonce(ctx)
	if err != nil {
		return PeerKey{}, err
	}

	return material.provenBy(nonce)
}

func (k PeerKey) Identity() PeerIdentity {
	return k.identity
}

func (k PeerKey) Hash() Hash {
	return k.identity.Hash()
}

func (k PeerKey) String() string {
	return k.material.String() + identityPartSeparator + nonceTextOf(k.identity.nonce)
}

func (k PeerKey) SignedSeedFrom(seed Seed, networkName string, issuedAt time.Time) Seed {
	seed.Hash = k.identity.Hash()
	seed.Identity = Some(k.identity)
	seed.AddressSignature = Some(k.addressSignatureOf(seed, networkName, issuedAt))

	return seed
}

func (k PeerKey) addressSignatureOf(
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
