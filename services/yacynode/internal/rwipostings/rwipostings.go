// Package rwipostings owns the RWI postings of this node and their held proofs,
// and is their only writer: callers read through PostingIndex and PostingsProofs,
// add page postings through PostingWriter, and drop through PostingPurger. A held
// proof leads a url's postings to its postings root. Vault keeps the projections
// handed to Open in step; each url keeps its words as one set.
package rwipostings

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PostingPurger interface {
	PurgePosting(
		tx *vault.Txn,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) (wasPurged bool, err error)
	PurgePostingsOfWordIn(
		tx *vault.Txn,
		word yacymodel.Hash,
		partition uint,
	) (amountOfPurgedPostings int, err error)
	PurgePostingsOf(tx *vault.Txn, url yacymodel.URLHash) (amountOfPurgedPostings int, err error)
}

type PostingIndex interface {
	PostingOf(
		tx *vault.Txn,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) (yacymodel.RWIPosting, bool, error)
	PostingsOfWordIn(
		tx *vault.Txn,
		word yacymodel.Hash,
		partition uint,
		visit func(yacymodel.RWIPosting) (bool, error),
	) error
}

type PostingWriter interface {
	Write(tx *vault.Txn, pagePostings yacymodel.PagePostings) error
}

type PostingsProofs interface {
	ProofsFor(
		ctx context.Context,
		offeredPostings []yacymodel.RWIPosting,
	) map[yacymodel.URLHash]yacymodel.PostingsProof
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func Open(
	ctx context.Context,
	vault *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
	rankedWords []yacymodel.Hash,
	urlWordsObserver vault.MemberSetObserver,
	projectionsOfConsumers []vault.Projection[yacymodel.RWIPosting],
) (PostingIndex, PostingWriter, PostingPurger, PostingsProofs, error) {
	postings, urlWordsIndex, err := registerPostings(
		ctx,
		vault,
		rankedWords,
		urlWordsObserver,
		projectionsOfConsumers,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	proofs, err := registerHeldProofs(vault)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	wordsPerURL := wordsPerURL{index: urlWordsIndex}
	heldProofs := heldProofs{proofs: proofs, wordsPerURL: wordsPerURL, postings: postings}
	directory := postingDirectory{
		postings:    postings,
		wordsPerURL: wordsPerURL,
		heldProofs:  heldProofs,
		partitions:  partitions,
	}

	return directory, directory, directory, postingsProofs{
		vault:      vault,
		heldProofs: heldProofs,
	}, nil
}

func PostingCodec() vault.ValueCodec[yacymodel.RWIPosting] {
	return postingValueCodec{}
}
