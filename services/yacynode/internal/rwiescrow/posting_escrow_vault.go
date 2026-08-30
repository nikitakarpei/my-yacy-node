package rwiescrow

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashcodec"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const (
	escrowedPostingBucket vault.Name = "rwi_escrow"
	postingHoldBucket     vault.Name = "rwi_escrow_expiry"
)

func registerPostingEscrow(
	v *vault.Vault,
) (*vault.Collection[postingIdentity, escrowedPosting], *vault.Set[postingHold], error) {
	escrowed, err := v.RegisterCollection(
		escrowedPostingBucket,
		escrowedPostingKeyCodec,
		escrowedPostingValueCodec{},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register escrowed postings: %w", err)
	}
	holds, err := v.RegisterSet(postingHoldBucket, postingHoldKeyCodec)
	if err != nil {
		return nil, nil, fmt.Errorf("register posting holds: %w", err)
	}

	return escrowed, holds, nil
}

var escrowedPostingKeyLayout = vault.PairKey(hashcodec.URLHash, hashcodec.Hash)

var escrowedPostingKeyCodec = escrowedPostingKeyLayout.KeyCodecFor(
	func(posting postingIdentity) (yacymodel.URLHash, yacymodel.Hash) {
		return posting.URL, posting.Word
	},
	func(url yacymodel.URLHash, word yacymodel.Hash) postingIdentity {
		return postingIdentity{Word: word, URL: url}
	},
)

func everyPostingWaitingFor(url yacymodel.URLHash) vault.KeyRange {
	return escrowedPostingKeyLayout.KeysWithFirst(url)
}

type escrowedPostingValueCodec struct{}

func (escrowedPostingValueCodec) Encode(escrowed escrowedPosting) ([]byte, error) {
	raw, err := rwipostings.PostingCodec().Encode(escrowed.Posting)
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

	posting, err := rwipostings.PostingCodec().Decode(raw[heldAtLength:])
	if err != nil {
		return escrowedPosting{}, fmt.Errorf("decode escrowed posting: %w", err)
	}

	return escrowedPosting{
		HeldAt:  time.Unix(seconds, nanoseconds).UTC(),
		Posting: posting,
	}, nil
}

var postingHoldKeyLayout = vault.TripleKey(vault.TimeKeyPart, hashcodec.Hash, hashcodec.URLHash)

var postingHoldKeyCodec = postingHoldKeyLayout.KeyCodecFor(
	func(hold postingHold) (time.Time, yacymodel.Hash, yacymodel.URLHash) {
		return hold.HeldAt, hold.Posting.Word, hold.Posting.URL
	},
	func(heldAt time.Time, word yacymodel.Hash, url yacymodel.URLHash) postingHold {
		return postingHold{HeldAt: heldAt, Posting: postingIdentity{Word: word, URL: url}}
	},
)

func everyHoldPlacedBefore(cutoff time.Time) vault.KeyRange {
	return postingHoldKeyLayout.KeysBeforeFirst(cutoff)
}
