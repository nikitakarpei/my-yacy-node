// Package rwipostings owns RWI posting storage and eviction. It is the only
// writer of postings: callers read through PostingIndex, add postings through
// PostingAdmitter, and drop them through PostingPurger. Projections learn
// through PostingObserver when a posting is stored and when a posting is
// purged. A posting that replaces a different posting of the same word and URL
// is purged and stored again; a posting that replaces an equal one is neither.
// Every port speaks the yacymodel vocabulary and lends cross-module work a
// shared transaction, so the schema never leaks; PostingCodec publishes the
// stored value codec for packages that hold a posting outside this index.
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
