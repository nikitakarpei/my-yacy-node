// Package rwipostings owns RWI posting storage, search, and eviction. It is the
// only writer of postings: callers read through PostingIndex, add postings through
// PostingAdmitter, and drop them through PostingPurger, while projections follow
// arrivals and departures through PostingObserver. Every port speaks the yacymodel
// vocabulary and lends cross-module work a shared transaction, so the schema never
// leaks; PostingForm publishes the stored value form for packages that hold a
// posting outside this index.
package rwipostings

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type PostingObserver interface {
	PostingStored(tx *vault.Txn, word yacymodel.Hash, url yacymodel.URLHash) error
	PostingPurged(tx *vault.Txn, word yacymodel.Hash, url yacymodel.URLHash) error
}

type PostingPurger interface {
	PurgePosting(tx *vault.Txn, word yacymodel.Hash, url yacymodel.URLHash) (bool, error)
}

type PostingIndex interface {
	RWICount(ctx context.Context) (int, error)
	PostingOf(
		ctx context.Context,
		word yacymodel.Hash,
		url yacymodel.URLHash,
	) (yacymodel.RWIPosting, bool, error)
	ScanWord(
		ctx context.Context,
		word yacymodel.Hash,
		visit func(yacymodel.RWIPosting) (bool, error),
	) error
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
		vault:     vault,
		postings:  postings,
		observers: postingObservers(observers),
	}

	return directory, directory, directory, nil
}

func PostingForm() vault.Codec[yacymodel.RWIPosting] {
	return postingCodec{}
}
