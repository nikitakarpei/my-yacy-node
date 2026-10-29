package yacymodel

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"fmt"
	"math"
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
	recoveryCommitment, err := recoveryCommitmentFrom(recoverySeeds)
	if err != nil {
		return keyMaterial{}, err
	}

	return keyMaterial{
		operationalKey:     ed25519.NewKeyFromSeed(operationalSeed[:]),
		recoverySeeds:      recoverySeeds,
		recoveryCommitment: recoveryCommitment,
	}, nil
}

func recoveryCommitmentFrom(recoverySeeds [recoverySeedsSize]byte) ([sha256.Size]byte, error) {
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

func (m keyMaterial) solvedNonce(ctx context.Context) (uint64, error) {
	keyDigest := keyDigestFrom(m.operationalPublicKey(), m.recoveryCommitment)
	var (
		searchOver  atomic.Bool
		solvedNonce uint64
		solved      bool
		searches    sync.WaitGroup
	)
	stopSearchOnCancel := context.AfterFunc(ctx, func() { searchOver.Store(true) })
	defer stopSearchOnCancel()
	searchCount := runtime.GOMAXPROCS(0)
	searchSpan := math.MaxUint64 / uint64(max(searchCount, 1))
	var searchStart uint64
	for range searchCount {
		firstNonce := searchStart
		searches.Go(func() {
			if nonce, found := nonceSearchedFrom(keyDigest, firstNonce, &searchOver); found {
				solvedNonce, solved = nonce, true
			}
		})
		searchStart += searchSpan
	}
	searches.Wait()
	if !solved {
		return 0, fmt.Errorf("solve the peer key: %w", context.Cause(ctx))
	}

	return solvedNonce, nil
}

func nonceSearchedFrom(
	keyDigest [sha256.Size]byte,
	firstNonce uint64,
	searchOver *atomic.Bool,
) (uint64, bool) {
	for nonce := firstNonce; !searchOver.Load(); nonce++ {
		if workOf(puzzleDigestFrom(keyDigest, nonce)) >= identityWorkMinimum &&
			searchOver.CompareAndSwap(false, true) {
			return nonce, true
		}
	}

	return 0, false
}

func (m keyMaterial) secretText() string {
	return Encode(m.operationalKey.Seed()) + identityPartSeparator + Encode(m.recoverySeeds[:])
}
