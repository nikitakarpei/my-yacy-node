package rwiescrow

import (
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
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
		escrowedPostingKeyLayout,
		escrowedPostingValueCodec{},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register escrowed postings: %w", err)
	}
	holds, err := v.RegisterSet(postingHoldBucket, postingHoldKeyLayout)
	if err != nil {
		return nil, nil, fmt.Errorf("register posting holds: %w", err)
	}

	return escrowed, holds, nil
}

var escrowedPostingKeyParts = vault.PairKey(hashkeypart.URLHash, hashkeypart.Hash)

var escrowedPostingKeyLayout = escrowedPostingKeyParts.KeyLayoutFor(
	func(posting postingIdentity) (yacymodel.URLHash, yacymodel.Hash) {
		return posting.URL, posting.Word
	},
	func(url yacymodel.URLHash, word yacymodel.Hash) postingIdentity {
		return postingIdentity{Word: word, URL: url}
	},
)

func everyPostingWaitingFor(url yacymodel.URLHash) vault.KeyRange {
	return escrowedPostingKeyParts.KeysWithFirst(url)
}

var errBadEscrowedPosting = errors.New("bad escrowed posting")

type escrowedPostingValueCodec struct{}

func (escrowedPostingValueCodec) Encode(escrowed escrowedPosting) ([]byte, error) {
	posting, err := rwipostings.PostingCodec().Encode(escrowed.Posting)
	if err != nil {
		return nil, fmt.Errorf("encode escrowed posting: %w", err)
	}

	var stored storedfields.Writer
	stored.Time(escrowed.HeldAt)
	stored.Bytes(posting)

	return stored.Record(), nil
}

func (escrowedPostingValueCodec) Decode(raw []byte) (escrowedPosting, error) {
	stored := storedfields.ReaderOf(raw, errBadEscrowedPosting)
	heldAt := stored.Time("held at")
	encodedPosting := stored.Bytes("posting")
	if err := stored.Err(); err != nil {
		return escrowedPosting{}, err
	}

	posting, err := rwipostings.PostingCodec().Decode(encodedPosting)
	if err != nil {
		return escrowedPosting{}, fmt.Errorf("decode escrowed posting: %w", err)
	}

	return escrowedPosting{HeldAt: heldAt, Posting: posting}, nil
}

var postingHoldKeyParts = vault.TripleKey(vault.TimeKeyPart, hashkeypart.Hash, hashkeypart.URLHash)

var postingHoldKeyLayout = postingHoldKeyParts.KeyLayoutFor(
	func(hold postingHold) (time.Time, yacymodel.Hash, yacymodel.URLHash) {
		return hold.HeldAt, hold.Posting.Word, hold.Posting.URL
	},
	func(heldAt time.Time, word yacymodel.Hash, url yacymodel.URLHash) postingHold {
		return postingHold{HeldAt: heldAt, Posting: postingIdentity{Word: word, URL: url}}
	},
)

func everyHoldPlacedBefore(cutoff time.Time) vault.KeyRange {
	return postingHoldKeyParts.KeysBeforeFirst(cutoff)
}
