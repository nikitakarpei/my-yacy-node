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

func NewPeerKey(ctx context.Context, searchTime time.Duration, clock Clock) (PeerKey, error) {
	return GeneratePeerKey(ctx, rand.Reader, searchTime, clock)
}

func GeneratePeerKey(
	ctx context.Context,
	entropy io.Reader,
	searchTime time.Duration,
	clock Clock,
) (PeerKey, error) {
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

	nonce, err := bestNonceFor(ctx, material.keyDigest(), searchTime, clock)
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

func (k PeerKey) Work() int {
	return k.identity.Work()
}

func (k PeerKey) WriteSecret(w io.Writer) error {
	secretText := k.material.secretText() + identityPartSeparator + nonceTextOf(k.identity.nonce)
	if _, err := io.WriteString(w, secretText); err != nil {
		return fmt.Errorf("write the peer key: %w", err)
	}

	return nil
}

func (k PeerKey) Format(state fmt.State, _ rune) {
	_, _ = fmt.Fprintf(state, "peer key of %s", k.Hash())
}

func (k PeerKey) SignedSeedFrom(seed Seed, networkName string, issuedAt time.Time) Seed {
	seed.Hash = k.identity.Hash()
	seed.Identity = Some(k.identity)
	seed.AddressSignature = Some(k.addressSignatureFrom(seed, networkName, issuedAt))

	return seed
}

func (k PeerKey) addressSignatureFrom(
	seed Seed,
	networkName string,
	issuedAt time.Time,
) AddressSignature {
	signature := AddressSignature{IssuedAt: issuedAt.UTC().Truncate(time.Second)}
	copy(signature.Signature[:], ed25519.Sign(
		k.material.operationalKey,
		addressClaimFrom(seed, networkName, signature.IssuedAt),
	))

	return signature
}
