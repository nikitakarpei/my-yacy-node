package rwiescrow

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const (
	heldBucket   vault.Name = "rwi_escrow"
	expiryBucket vault.Name = "rwi_escrow_expiry"
)

const heldPostingBytes = 256

var (
	heldKeyLayout   = vaultkey.Pair(vaultkey.Text, vaultkey.Text)
	expiryKeyLayout = vaultkey.Triple(vaultkey.Time, vaultkey.Text, vaultkey.Text)
)

type postingIdentity struct {
	Word yacymodel.Hash
	URL  yacymodel.URLHash
}

type postingHold struct {
	HeldAt  time.Time
	Posting postingIdentity
}

func registerHeldPostings(
	v *vault.Vault,
) (*vault.Collection[heldPosting], *vault.Set, error) {
	held, err := vault.Register(v, heldBucket, heldPostingCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register held postings: %w", err)
	}
	expiry, err := vault.RegisterSet(v, expiryBucket)
	if err != nil {
		return nil, nil, fmt.Errorf("register held posting expiry: %w", err)
	}

	return held, expiry, nil
}

func heldKey(posting postingIdentity) vault.Key {
	return heldKeyLayout.Key(posting.URL.String(), posting.Word.String()).Bytes()
}

func heldKeyPrefixOfURL(url yacymodel.URLHash) vault.Key {
	return heldKeyLayout.First(url.String()).Bytes()
}

func parseHeldKey(key vault.Key) (postingIdentity, error) {
	url, word, err := heldKeyLayout.Parts(vaultkey.KeyFrom(key))
	if err != nil {
		return postingIdentity{}, fmt.Errorf("held posting key: %w", err)
	}

	return parseIdentity(word, url)
}

func parseIdentity(word string, url string) (postingIdentity, error) {
	parsedWord, err := yacymodel.ParseHash(word)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("held posting word hash: %w", err)
	}
	parsedURL, err := yacymodel.ParseURLHash(url)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("held posting url hash: %w", err)
	}

	return postingIdentity{Word: parsedWord, URL: parsedURL}, nil
}

func expiryKey(heldAt time.Time, posting postingIdentity) vault.Key {
	return expiryKeyLayout.Key(heldAt, posting.Word.String(), posting.URL.String()).Bytes()
}

func parseExpiryKey(key vault.Key) (postingHold, error) {
	heldAt, word, url, err := expiryKeyLayout.Parts(vaultkey.KeyFrom(key))
	if err != nil {
		return postingHold{}, fmt.Errorf("held posting expiry key: %w", err)
	}
	posting, err := parseIdentity(word, url)
	if err != nil {
		return postingHold{}, err
	}

	return postingHold{HeldAt: heldAt, Posting: posting}, nil
}
