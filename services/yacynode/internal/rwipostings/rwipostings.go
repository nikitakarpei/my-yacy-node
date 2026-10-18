// Package rwipostings owns the RWI postings of this node, and is their only
// writer: callers read through PostingIndex, add through PostingWriter, and
// drop through PostingPurger, one posting or every posting of a url at once.
// Vault keeps the projections handed to Open in step. Each url keeps its words
// as one set, where a word the ranking lists takes fewer bytes. PostingCodec
// publishes the stored value codec of a posting.
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
	Write(tx *vault.Txn, postings ...yacymodel.RWIPosting) error
}

//nolint:revive // argument-limit: independently-meaningful collaborators, not configuration to bundle further
func Open(
	ctx context.Context,
	vault *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
	rankedWords []yacymodel.Hash,
	urlWordsObserver vault.MemberSetObserver,
	projections []vault.Projection[yacymodel.RWIPosting],
) (PostingIndex, PostingWriter, PostingPurger, error) {
	postings, urlWords, err := registerPostings(
		ctx,
		vault,
		rankedWords,
		urlWordsObserver,
		projections,
	)
	if err != nil {
		return nil, nil, nil, err
	}

	directory := postingDirectory{
		postings:   postings,
		urlWords:   urlWords,
		partitions: partitions,
	}

	return directory, directory, directory, nil
}

func PostingCodec() vault.ValueCodec[yacymodel.RWIPosting] {
	return postingValueCodec{}
}
