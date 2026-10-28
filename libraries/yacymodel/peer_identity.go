package yacymodel

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"strings"
)

const identityWorkMinimum = 28

const identityDomain = "yacy-peer-id/1"

const identityPartSeparator = ":"

var ErrBadPeerIdentity = errors.New("bad peer identity")

type PeerIdentity struct {
	operationalKey     [ed25519.PublicKeySize]byte
	recoveryCommitment [sha256.Size]byte
	nonce              uint64
}

func ParsePeerIdentity(text string) (PeerIdentity, error) {
	parts, separated := partsOf(text)
	if !separated {
		return PeerIdentity{}, fmt.Errorf("%w: want 3 parts", ErrBadPeerIdentity)
	}

	var (
		operationalKey     [ed25519.PublicKeySize]byte
		recoveryCommitment [sha256.Size]byte
	)
	if err := decodeExactly(parts[0], operationalKey[:]); err != nil {
		return PeerIdentity{}, fmt.Errorf("%w: operational key: %w", ErrBadPeerIdentity, err)
	}
	if err := decodeExactly(parts[1], recoveryCommitment[:]); err != nil {
		return PeerIdentity{}, fmt.Errorf("%w: recovery commitment: %w", ErrBadPeerIdentity, err)
	}
	nonce, err := nonceOf(parts[2])
	if err != nil {
		return PeerIdentity{}, fmt.Errorf("%w: nonce: %w", ErrBadPeerIdentity, err)
	}

	return provenIdentityFrom(operationalKey, recoveryCommitment, nonce)
}

func partsOf(text string) ([3]string, bool) {
	parts := strings.Split(text, identityPartSeparator)
	if len(parts) != 3 {
		return [3]string{}, false
	}

	return [3]string(parts), true
}

func decodeExactly(text string, raw []byte) error {
	decoded, err := Decode(text)
	if err != nil {
		return err
	}
	if len(decoded) != len(raw) || Encode(decoded) != text {
		return fmt.Errorf("%w: not the encoding of %d bytes", ErrInvalidBase64, len(raw))
	}
	copy(raw, decoded)

	return nil
}

func nonceOf(text string) (uint64, error) {
	var nonce [8]byte
	if err := decodeExactly(text, nonce[:]); err != nil {
		return 0, err
	}

	return binary.BigEndian.Uint64(nonce[:]), nil
}

func provenIdentityFrom(
	operationalKey [ed25519.PublicKeySize]byte,
	recoveryCommitment [sha256.Size]byte,
	nonce uint64,
) (PeerIdentity, error) {
	identity := PeerIdentity{
		operationalKey:     operationalKey,
		recoveryCommitment: recoveryCommitment,
		nonce:              nonce,
	}
	if work := workOf(identity.digest()); work < identityWorkMinimum {
		return PeerIdentity{}, fmt.Errorf("%w: work %d, want at least %d",
			ErrBadPeerIdentity, work, identityWorkMinimum)
	}

	return identity, nil
}

func workOf(digest [sha256.Size]byte) int {
	return bits.TrailingZeros64(binary.BigEndian.Uint64(digest[sha256.Size-8:]))
}

func (id PeerIdentity) digest() [sha256.Size]byte {
	return puzzleDigestOf(keyDigestOf(id.operationalKey, id.recoveryCommitment), id.nonce)
}

func puzzleDigestOf(keyDigest [sha256.Size]byte, nonce uint64) [sha256.Size]byte {
	var input [sha256.Size + 8]byte
	copy(input[:], keyDigest[:])
	binary.BigEndian.PutUint64(input[sha256.Size:], nonce)

	return sha256.Sum256(input[:])
}

func keyDigestOf(
	operationalKey [ed25519.PublicKeySize]byte,
	recoveryCommitment [sha256.Size]byte,
) [sha256.Size]byte {
	keyDigest := sha256.New()
	keyDigest.Write([]byte(identityDomain))
	keyDigest.Write(operationalKey[:])
	keyDigest.Write(recoveryCommitment[:])

	return [sha256.Size]byte(keyDigest.Sum(nil))
}

func (id PeerIdentity) Hash() Hash {
	digest := id.digest()

	return Hash{raw: [HashByteLength]byte(digest[:HashByteLength]), present: true}
}

func (id PeerIdentity) String() string {
	return Encode(id.operationalKey[:]) + identityPartSeparator +
		Encode(id.recoveryCommitment[:]) + identityPartSeparator +
		nonceTextOf(id.nonce)
}

func nonceTextOf(nonce uint64) string {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], nonce)

	return Encode(raw[:])
}

func (id PeerIdentity) verifies(claim, signature []byte) bool {
	return ed25519.Verify(id.operationalKey[:], claim, signature)
}
