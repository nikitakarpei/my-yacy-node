package yacymodel

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"time"
)

var ErrBadPeerKey = errors.New("bad peer key")

type PeerKey struct {
	identity PeerIdentity
	secret   peerSecret
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

	secret := peerSecretFrom(operationalSeed, recoverySeeds)
	recoveryCommitment, err := secret.recoveryCommitment()
	if err != nil {
		return PeerKey{}, err
	}

	return peerKeyFrom(secret, recoveryCommitment, nonce)
}

func peerKeyFrom(
	secret peerSecret,
	recoveryCommitment [sha256.Size]byte,
	nonce uint64,
) (PeerKey, error) {
	identity, err := provenIdentityFrom(secret.operationalPublicKey(), recoveryCommitment, nonce)
	if err != nil {
		return PeerKey{}, fmt.Errorf("%w: %w", ErrBadPeerKey, err)
	}

	return PeerKey{identity: identity, secret: secret}, nil
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

	secret := peerSecretFrom(
		[ed25519.SeedSize]byte(seeds[:ed25519.SeedSize]),
		[recoverySeedsSize]byte(seeds[ed25519.SeedSize:]),
	)
	recoveryCommitment, err := secret.recoveryCommitment()
	if err != nil {
		return PeerKey{}, err
	}
	keyDigest := keyDigestFrom(secret.operationalPublicKey(), recoveryCommitment)
	nonce, err := bestNonceFor(ctx, keyDigest, searchTime, clock)
	if err != nil {
		return PeerKey{}, err
	}

	return peerKeyFrom(secret, recoveryCommitment, nonce)
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
	secretText := k.secret.text() + identityPartSeparator + nonceTextOf(k.identity.nonce)
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

func (k PeerKey) EndorsementOf(tlsPublicKey []byte, networkName string) TLSKeyEndorsement {
	return TLSKeyEndorsement{
		Identity:  k.identity,
		Signature: k.secret.signatureFrom(tlsKeyClaimFrom(k.Hash(), tlsPublicKey, networkName)),
	}
}

func (k PeerKey) addressSignatureFrom(
	seed Seed,
	networkName string,
	issuedAt time.Time,
) AddressSignature {
	signedAt := issuedAt.UTC().Truncate(time.Second)

	return AddressSignature{
		IssuedAt:  signedAt,
		Signature: k.secret.signatureFrom(addressClaimFrom(seed, networkName, signedAt)),
	}
}
