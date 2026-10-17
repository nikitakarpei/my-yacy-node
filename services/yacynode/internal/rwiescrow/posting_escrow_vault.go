package rwiescrow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

const (
	escrowedPostingBucket vault.Name = "rwi_escrow"
	postingHoldBucket     vault.Name = "rwi_escrow_expiry"
)

const postingHoldsVersion = "2"

func registerPostingEscrow(
	ctx context.Context,
	v *vault.Vault,
) (
	*vault.Collection[postingIdentity, escrowedPosting],
	*vault.ValueIndex[postingHold, postingHold],
	error,
) {
	holds, err := v.RegisterValueIndex(
		postingHoldBucket,
		postingHoldsVersion,
		postingHoldKeyLayout,
		func(hold postingHold) postingHold { return hold },
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register posting holds: %w", err)
	}
	escrowed, err := v.RegisterProjectedCollection(
		ctx,
		escrowedPostingBucket,
		escrowedPostingKeyLayout,
		escrowedPostingValueCodec{},
		vault.ProjectedAs(postingHoldOf, holds),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register escrowed postings: %w", err)
	}

	return escrowed, holds, nil
}

func postingHoldOf(identity postingIdentity, escrowed escrowedPosting) postingHold {
	return postingHold{HeldAt: escrowed.HeldAt, Posting: identity}
}

var pageVersionKeyPart = vault.BytesKeyPartFrom(pagefingerprint.AppendBytes, pageVersionFrom)

func pageVersionFrom(stored vault.StoredBytes) (pagefingerprint.Fingerprint, error) {
	return pagefingerprint.ParseBytes(stored.AppendTo(nil))
}

var escrowedPostingKeyParts = vault.TripleKey(
	hashkeypart.URLHash,
	hashkeypart.Hash,
	pageVersionKeyPart,
)

var escrowedPostingKeyLayout = escrowedPostingKeyParts.KeyLayoutFor(
	func(
		posting postingIdentity,
	) (yacymodel.URLHash, yacymodel.Hash, pagefingerprint.Fingerprint) {
		return posting.URL, posting.Word, posting.Version
	},
	func(
		url yacymodel.URLHash,
		word yacymodel.Hash,
		version pagefingerprint.Fingerprint,
	) postingIdentity {
		return postingIdentity{Word: word, URL: url, Version: version}
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
	rwiorigin.Write(&stored, escrowed.Origin)

	return stored.Record(), nil
}

func (escrowedPostingValueCodec) Decode(raw []byte) (escrowedPosting, error) {
	stored := storedfields.ReaderOf(raw, errBadEscrowedPosting)
	heldAt := stored.Time("held at")
	encodedPosting := stored.Bytes("posting")
	origin := rwiorigin.From(stored)
	if err := stored.Err(); err != nil {
		return escrowedPosting{}, err
	}

	posting, err := rwipostings.PostingCodec().Decode(encodedPosting)
	if err != nil {
		return escrowedPosting{}, fmt.Errorf("decode escrowed posting: %w", err)
	}

	return escrowedPosting{HeldAt: heldAt, Origin: origin, Posting: posting}, nil
}

var postingHoldKeyParts = vault.QuadKey(
	vault.TimeKeyPart,
	hashkeypart.Hash,
	hashkeypart.URLHash,
	pageVersionKeyPart,
)

var postingHoldKeyLayout = postingHoldKeyParts.KeyLayoutFor(
	func(
		hold postingHold,
	) (time.Time, yacymodel.Hash, yacymodel.URLHash, pagefingerprint.Fingerprint) {
		return hold.HeldAt, hold.Posting.Word, hold.Posting.URL, hold.Posting.Version
	},
	func(
		heldAt time.Time,
		word yacymodel.Hash,
		url yacymodel.URLHash,
		version pagefingerprint.Fingerprint,
	) postingHold {
		return postingHold{
			HeldAt:  heldAt,
			Posting: postingIdentity{Word: word, URL: url, Version: version},
		}
	},
)

func everyHoldPlacedBefore(cutoff time.Time) vault.KeyRange {
	return postingHoldKeyParts.KeysBeforeFirst(cutoff)
}
