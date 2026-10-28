package yacymodel

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudflare/circl/sign/slhdsa"
)

const recoveryScheme = slhdsa.SHA2_128s

const recoverySeedsSize = 48

var ErrBadPeerKey = errors.New("bad peer key")

type PeerKey struct {
	operationalKey ed25519.PrivateKey
	recoverySeeds  [recoverySeedsSize]byte
	identity       PeerIdentity
}

func ParsePeerKey(text string) (PeerKey, error) {
	parts := strings.Split(text, identityPartSeparator)
	if len(parts) != 3 {
		return PeerKey{}, fmt.Errorf("%w: %d parts, want 3", ErrBadPeerKey, len(parts))
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

	unproven, err := unprovenPeerKeyFrom(operationalSeed, recoverySeeds)
	if err != nil {
		return PeerKey{}, err
	}

	return unproven.provenBy(nonce)
}

func unprovenPeerKeyFrom(
	operationalSeed [ed25519.SeedSize]byte,
	recoverySeeds [recoverySeedsSize]byte,
) (PeerKey, error) {
	operationalKey := ed25519.NewKeyFromSeed(operationalSeed[:])
	recoveryCommitment, err := recoveryCommitmentOf(recoverySeeds)
	if err != nil {
		return PeerKey{}, err
	}

	return PeerKey{
		operationalKey: operationalKey,
		recoverySeeds:  recoverySeeds,
		identity: PeerIdentity{
			operationalKey: [ed25519.PublicKeySize]byte(
				operationalKey.Public().(ed25519.PublicKey),
			),
			recoveryCommitment: recoveryCommitment,
		},
	}, nil
}

func recoveryCommitmentOf(recoverySeeds [recoverySeedsSize]byte) ([sha256.Size]byte, error) {
	recoveryPublicKey, _, err := slhdsa.GenerateKey(
		bytes.NewReader(recoverySeeds[:]),
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

func (k PeerKey) provenBy(nonce uint64) (PeerKey, error) {
	k.identity.nonce = nonce
	identity, err := k.identity.proven()
	if err != nil {
		return PeerKey{}, fmt.Errorf("%w: %w", ErrBadPeerKey, err)
	}
	k.identity = identity

	return k, nil
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

	unproven, err := unprovenPeerKeyFrom(operationalSeed, recoverySeeds)
	if err != nil {
		return PeerKey{}, err
	}

	return unproven.provenBy(solvedNonceFor(unproven.identity))
}

func solvedNonceFor(identity PeerIdentity) uint64 {
	keyDigest := keyDigestOf(identity.operationalKey, identity.recoveryCommitment)
	var (
		solved      atomic.Bool
		solvedNonce uint64
		solvedOnce  sync.Once
		searches    sync.WaitGroup
	)
	for search := range runtime.GOMAXPROCS(0) {
		searches.Go(func() {
			nonce, found := nonceSearchedFrom(keyDigest, uint64(search)<<56, &solved)
			if found {
				solvedOnce.Do(func() { solvedNonce = nonce })
			}
		})
	}
	searches.Wait()

	return solvedNonce
}

func nonceSearchedFrom(
	keyDigest [sha256.Size]byte,
	firstNonce uint64,
	solved *atomic.Bool,
) (uint64, bool) {
	for nonce := firstNonce; !solved.Load(); nonce++ {
		if workOf(puzzleDigestOf(keyDigest, nonce)) >= identityWorkMinimum {
			solved.Store(true)

			return nonce, true
		}
	}

	return 0, false
}

func (k PeerKey) Identity() PeerIdentity {
	return k.identity
}

func (k PeerKey) String() string {
	var nonce [8]byte
	binary.BigEndian.PutUint64(nonce[:], k.identity.nonce)

	return Encode(k.operationalKey.Seed()) + identityPartSeparator +
		Encode(k.recoverySeeds[:]) + identityPartSeparator +
		Encode(nonce[:])
}

func (k PeerKey) AddressSignatureOf(
	seed Seed,
	networkName string,
	issuedAt time.Time,
) AddressSignature {
	signature := AddressSignature{IssuedAt: issuedAt.UTC().Truncate(time.Second)}
	copy(signature.Signature[:], ed25519.Sign(
		k.operationalKey,
		addressClaimOf(seed, networkName, signature.IssuedAt),
	))

	return signature
}
