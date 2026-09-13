// Package rwipostings owns the RWI postings of this node, and is their only
// writer: callers read through PostingIndex, add through PostingAdmitter, and
// drop through PostingPurger, while projections follow through PostingObserver.
// PostingCodec publishes the stored value codec of a posting.
package rwipostings

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PostingObserver interface {
	PostingStored(tx *vault.Txn, posting yacymodel.RWIPosting) error
	PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error
}

type PostingPurger interface {
	PurgePosting(
		tx *vault.Txn,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) (wasPurged bool, err error)
}

type PostingIndex interface {
	RWICount(tx *vault.Txn) (int, error)
	PostingOf(
		tx *vault.Txn,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) (yacymodel.RWIPosting, bool, error)
}

type PostingAdmitter interface {
	Admit(tx *vault.Txn, posting yacymodel.RWIPosting) error
}

func Open(
	vault *vault.Vault,
	observers ...PostingObserver,
) (PostingIndex, PostingAdmitter, PostingPurger, error) {
	postings, err := registerPostings(vault)
	if err != nil {
		return nil, nil, nil, err
	}

	directory := postingDirectory{
		postings:  postings,
		observers: postingObservers(observers),
	}

	return directory, directory, directory, nil
}

func PostingCodec() vault.ValueCodec[yacymodel.RWIPosting] {
	return postingValueCodec{}
}
