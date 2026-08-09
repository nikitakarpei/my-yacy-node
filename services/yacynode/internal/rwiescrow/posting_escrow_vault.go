package rwiescrow

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
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

var escrowedPostingKeyLayout = vaultkey.Pair(hashcodec.URLHash, hashcodec.Hash)

type escrowedPostingKeyCodec struct{}

func (escrowedPostingKeyCodec) Encode(posting postingIdentity) vaultkey.Key {
	return escrowedPostingKeyLayout.Key(posting.URL, posting.Word)
}

func (escrowedPostingKeyCodec) Decode(storedKey []byte) (postingIdentity, error) {
	url, word, err := escrowedPostingKeyLayout.Parts(storedKey)
	if err != nil {
		return postingIdentity{}, fmt.Errorf("escrowed posting key: %w", err)
	}

	return postingIdentity{Word: word, URL: url}, nil
}

func escrowedPostingKeysOf(url yacymodel.URLHash) vaultkey.KeyRange {
	return escrowedPostingKeyLayout.KeysWithFirst(url)
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

var postingHoldKeyLayout = vaultkey.Triple(vaultkey.Time, hashcodec.Hash, hashcodec.URLHash)

type postingHoldKeyCodec struct{}

func (postingHoldKeyCodec) Encode(hold postingHold) vaultkey.Key {
	return postingHoldKeyLayout.Key(hold.HeldAt, hold.Posting.Word, hold.Posting.URL)
}

func (postingHoldKeyCodec) Decode(storedKey []byte) (postingHold, error) {
	heldAt, word, url, err := postingHoldKeyLayout.Parts(storedKey)
	if err != nil {
		return postingHold{}, fmt.Errorf("posting hold key: %w", err)
	}

	return postingHold{HeldAt: heldAt, Posting: postingIdentity{Word: word, URL: url}}, nil
}

func postingHoldKeysBefore(cutoff time.Time) vaultkey.KeyRange {
	return postingHoldKeyLayout.KeysBeforeFirst(cutoff)
}
