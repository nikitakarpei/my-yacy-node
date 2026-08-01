package rwiescrow

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

const (
	heldBucket   vault.Name = "rwi_escrow"
	expiryBucket vault.Name = "rwi_escrow_expiry"
)

const heldAtDigits = 20

const heldPostingBytes = 256

type postingIdentity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

func registerHeldPostings(
	v *vault.Vault,
) (*vault.Collection[heldPosting], *vault.Collection[struct{}], error) {
	held, err := vault.Register(v, heldBucket, heldPostingCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register held postings: %w", err)
	}
	expiry, err := vault.Register(v, expiryBucket, presenceCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register held posting expiry: %w", err)
	}

	return held, expiry, nil
}

func heldKey(posting postingIdentity) vault.Key {
	key := make(vault.Key, 0, yacymodel.HashLength*2)
	key = append(key, posting.URL.String()...)
	key = append(key, posting.Word.String()...)

	return key
}

func parseHeldKey(key vault.Key) (postingIdentity, error) {
	wantLength := yacymodel.HashLength * 2
	if len(key) != wantLength {
		return postingIdentity{},
			fmt.Errorf("held posting key: length %d, want %d", len(key), wantLength)
	}

	return parseIdentity(key)
}

func expiryKey(heldAt time.Time, posting postingIdentity) vault.Key {
	key := make(vault.Key, 0, heldAtDigits+yacymodel.HashLength*2)
	key = append(key, heldAtPrefix(heldAt)...)
	key = append(key, posting.URL.String()...)
	key = append(key, posting.Word.String()...)

	return key
}

func parseExpiryKey(key vault.Key) (postingIdentity, error) {
	wantLength := heldAtDigits + yacymodel.HashLength*2
	if len(key) != wantLength {
		return postingIdentity{},
			fmt.Errorf("held posting expiry key: length %d, want %d", len(key), wantLength)
	}

	return parseIdentity(key[heldAtDigits:])
}

func parseIdentity(raw []byte) (postingIdentity, error) {
	url, err := yacymodel.ParseURLHash(string(raw[:yacymodel.HashLength]))
	if err != nil {
		return postingIdentity{}, fmt.Errorf("held posting url hash: %w", err)
	}
	word, err := yacymodel.ParseHash(string(raw[yacymodel.HashLength:]))
	if err != nil {
		return postingIdentity{}, fmt.Errorf("held posting word hash: %w", err)
	}

	return postingIdentity{Word: word, URL: url}, nil
}

func heldAtPrefix(at time.Time) vault.Key {
	return fmt.Appendf(make(vault.Key, 0, heldAtDigits), "%0*d", heldAtDigits, at.UnixNano())
}
