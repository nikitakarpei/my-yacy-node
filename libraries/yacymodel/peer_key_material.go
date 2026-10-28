package yacymodel

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/cloudflare/circl/sign/slhdsa"
)

const recoveryScheme = slhdsa.SHA2_128s

const recoverySeedsSize = 48

type keyMaterial struct {
	operationalKey     ed25519.PrivateKey
	recoverySeeds      [recoverySeedsSize]byte
	recoveryCommitment [sha256.Size]byte
}

func keyMaterialFrom(
	operationalSeed [ed25519.SeedSize]byte,
	recoverySeeds [recoverySeedsSize]byte,
) (keyMaterial, error) {
	recoveryCommitment, err := recoveryCommitmentOf(recoverySeeds)
	if err != nil {
		return keyMaterial{}, err
	}

	return keyMaterial{
		operationalKey:     ed25519.NewKeyFromSeed(operationalSeed[:]),
		recoverySeeds:      recoverySeeds,
		recoveryCommitment: recoveryCommitment,
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

func (m keyMaterial) provenBy(nonce uint64) (PeerKey, error) {
	identity, err := provenIdentityFrom(m.operationalPublicKey(), m.recoveryCommitment, nonce)
	if err != nil {
		return PeerKey{}, fmt.Errorf("%w: %w", ErrBadPeerKey, err)
	}

	return PeerKey{material: m, identity: identity}, nil
}

func (m keyMaterial) operationalPublicKey() [ed25519.PublicKeySize]byte {
	return [ed25519.PublicKeySize]byte(m.operationalKey.Public().(ed25519.PublicKey))
}

func (m keyMaterial) solvedNonce() uint64 {
	keyDigest := keyDigestOf(m.operationalPublicKey(), m.recoveryCommitment)
	var (
		solved      atomic.Bool
		solvedNonce uint64
		solvedOnce  sync.Once
		searches    sync.WaitGroup
		firstNonce  uint64
	)
	for range runtime.GOMAXPROCS(0) {
		searchStart := firstNonce
		searches.Go(func() {
			nonce, found := nonceSearchedFrom(keyDigest, searchStart, &solved)
			if found {
				solvedOnce.Do(func() { solvedNonce = nonce })
			}
		})
		firstNonce += 1 << 56
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

func (m keyMaterial) String() string {
	return Encode(m.operationalKey.Seed()) + identityPartSeparator + Encode(m.recoverySeeds[:])
}
