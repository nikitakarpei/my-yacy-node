// Package rwi owns RWI posting intake, storage, search, and eviction. It is the
// only writer of postings: callers read through PostingIndex, hand postings in
// through PostingReceiver, and drop them through PostingPurger, while projections
// follow arrivals and departures through PostingObserver. Every port speaks the
// yacymodel vocabulary and lends cross-module work a shared transaction, so the
// schema never leaks.
package rwipostings

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

type PostingObserver interface {
	PostingStored(tx *vault.Txn, word, url yacymodel.Hash) error
	PostingPurged(tx *vault.Txn, word, url yacymodel.Hash) error
}

type PostingPurger interface {
	PurgePosting(tx *vault.Txn, word, url yacymodel.Hash) (bool, error)
}

type PostingIndex interface {
	RWICount(ctx context.Context) (int, error)
	ScanWord(
		ctx context.Context,
		word yacymodel.Hash,
		visit func(yacymodel.RWIPosting) (bool, error),
	) error
}

type PostingReceiver interface {
	Receive(ctx context.Context, entries []yacymodel.RWIPosting) (Receipt, error)
}

type Receipt struct {
	Busy       bool
	TooLarge   bool
	Pause      int
	UnknownURL []yacymodel.Hash
}

type Config struct {
	BatchCap     int
	PauseSeconds int
}

func Open(
	vault *vault.Vault,
	urls urlmeta.URLDirectory,
	cfg Config,
	observers ...PostingObserver,
) (PostingIndex, PostingReceiver, PostingPurger, error) {
	postings, err := registerPostings(vault)
	if err != nil {
		return nil, nil, nil, err
	}

	watched := postingObservers(observers)
	directory := postingDirectory{vault: vault, postings: postings, observers: watched}
	intake := postingIntake{
		vault:        vault,
		postings:     postings,
		observers:    watched,
		urls:         urls,
		batchCap:     cfg.BatchCap,
		pauseSeconds: cfg.PauseSeconds,
	}

	return directory, intake, directory, nil
}
