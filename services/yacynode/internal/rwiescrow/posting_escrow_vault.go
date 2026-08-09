package rwiescrow

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vaultkey"
)

const (
	escrowedPostingBucket vault.Name = "rwi_escrow"
	postingHoldBucket     vault.Name = "rwi_escrow_expiry"
)

func registerPostingEscrow(
	v *vault.Vault,
) (*vault.Collection[postingIdentity, escrowedPosting], *vault.Set[postingHold], error) {
	escrowed, err := vault.RegisterCollection(
		v,
		escrowedPostingBucket,
		escrowedPostingKeyCodec{},
		escrowedPostingValueCodec{},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register escrowed postings: %w", err)
	}
	holds, err := vault.RegisterSet(v, postingHoldBucket, postingHoldKeyCodec{})
	if err != nil {
		return nil, nil, fmt.Errorf("register posting holds: %w", err)
	}

	return escrowed, holds, nil
}

var escrowedPostingKeyLayout = vaultkey.Pair(vaultkey.Text, vaultkey.Text)

type escrowedPostingKeyCodec struct{}

func (escrowedPostingKeyCodec) Encode(posting postingIdentity) vaultkey.Key {
	return escrowedPostingKeyLayout.Key(posting.URL.String(), posting.Word.String())
}

func (escrowedPostingKeyCodec) Decode(key vaultkey.Key) (postingIdentity, error) {
	url, word, err := escrowedPostingKeyLayout.Parts(key)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("escrowed posting key: %w", err)
	}

	return parsedIdentity(word, url)
}

func parsedIdentity(word string, url string) (postingIdentity, error) {
	parsedWord, err := yacymodel.ParseHash(word)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("posting word hash: %w", err)
	}
	parsedURL, err := yacymodel.ParseURLHash(url)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("posting url hash: %w", err)
	}

	return postingIdentity{Word: parsedWord, URL: parsedURL}, nil
}

func escrowedPostingKeysOf(url yacymodel.URLHash) vaultkey.KeyRange {
	return vaultkey.KeysUnder(escrowedPostingKeyLayout.First(url.String()))
}

type escrowedPostingValueCodec struct{}

func (escrowedPostingValueCodec) Encode(escrowed escrowedPosting) ([]byte, error) {
	raw, err := rwipostings.PostingForm().Encode(escrowed.Posting)
	if err != nil {
		return nil, fmt.Errorf("encode escrowed posting: %w", err)
	}

	heldAt, err := binary.Append(nil, binary.BigEndian, []int64{
		escrowed.HeldAt.Unix(),
		int64(escrowed.HeldAt.Nanosecond()),
	})
	if err != nil {
		return nil, fmt.Errorf("encode escrowed posting hold time: %w", err)
	}

	return append(heldAt, raw...), nil
}

func (escrowedPostingValueCodec) Decode(raw []byte) (escrowedPosting, error) {
	heldAt := make([]int64, 2)
	heldAtLength, err := binary.Decode(raw, binary.BigEndian, heldAt)
	if err != nil {
		return escrowedPosting{}, fmt.Errorf("escrowed posting hold time: %w", err)
	}
	seconds, nanoseconds := heldAt[0], heldAt[1]

	posting, err := rwipostings.PostingForm().Decode(raw[heldAtLength:])
	if err != nil {
		return escrowedPosting{}, fmt.Errorf("decode escrowed posting: %w", err)
	}

	return escrowedPosting{
		HeldAt:  time.Unix(seconds, nanoseconds).UTC(),
		Posting: posting,
	}, nil
}

var postingHoldKeyLayout = vaultkey.Triple(vaultkey.Time, vaultkey.Text, vaultkey.Text)

type postingHoldKeyCodec struct{}

func (postingHoldKeyCodec) Encode(hold postingHold) vaultkey.Key {
	return postingHoldKeyLayout.Key(
		hold.HeldAt,
		hold.Posting.Word.String(),
		hold.Posting.URL.String(),
	)
}

func (postingHoldKeyCodec) Decode(key vaultkey.Key) (postingHold, error) {
	heldAt, word, url, err := postingHoldKeyLayout.Parts(key)
	if err != nil {
		return postingHold{}, fmt.Errorf("posting hold key: %w", err)
	}
	posting, err := parsedIdentity(word, url)
	if err != nil {
		return postingHold{}, err
	}

	return postingHold{HeldAt: heldAt, Posting: posting}, nil
}

func postingHoldKeysBefore(cutoff time.Time) vaultkey.KeyRange {
	return vaultkey.KeysBefore(postingHoldKeyLayout.First(cutoff))
}
