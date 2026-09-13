package rwipostings

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingIdentity struct {
	word yacymodel.Hash
	url  yacymodel.URLHash
}

type postingDirectory struct {
	postings  *vault.Collection[postingIdentity, yacymodel.RWIPosting]
	observers postingObservers
}

func (d postingDirectory) RWICount(tx *vault.Txn) (int, error) {
	return collectionLength(tx, d.postings)
}

func (d postingDirectory) PostingOf(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	key := postingIdentity{word: word, url: url}
	stored, found, err := d.postings.Get(tx, key)
	if err != nil {
		return yacymodel.RWIPosting{}, false, fmt.Errorf("read rwi posting: %w", err)
	}
	if !found {
		return yacymodel.RWIPosting{}, false, nil
	}

	return postingFrom(key, stored), true, nil
}

func postingFrom(key postingIdentity, stored yacymodel.RWIPosting) yacymodel.RWIPosting {
	stored.WordHash = key.word
	stored.URLHash = key.url

	return stored
}

func (d postingDirectory) Admit(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	key := postingIdentity{word: posting.WordHash, url: posting.URLHash}
	previousPosting, postingWasHeld, err := d.postings.Put(tx, key, posting)
	if err != nil {
		return fmt.Errorf("store rwi posting: %w", err)
	}
	if !postingWasHeld {
		return d.observers.stored(tx, posting)
	}

	return d.observers.updated(tx, postingFrom(key, previousPosting), posting)
}

func (d postingDirectory) PurgePosting(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (bool, error) {
	key := postingIdentity{word: word, url: url}
	purgedPosting, postingWasHeld, err := d.postings.Delete(tx, key)
	if err != nil {
		return false, fmt.Errorf("delete rwi posting: %w", err)
	}
	if !postingWasHeld {
		return false, nil
	}
	if err := d.observers.purged(tx, postingFrom(key, purgedPosting)); err != nil {
		return false, err
	}

	return true, nil
}

func (d postingDirectory) ScanWord(
	ctx context.Context,
	tx *vault.Txn,
	word yacymodel.Hash,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	err := d.postings.Scan(
		tx,
		everyPostingOf(word),
		func(identity postingIdentity, entry yacymodel.RWIPosting) (bool, error) {
			if err := ctx.Err(); err != nil {
				return false, fmt.Errorf("context: %w", err)
			}

			return visit(postingFrom(identity, entry))
		},
	)
	if err != nil {
		return fmt.Errorf("scan word postings: %w", err)
	}

	return nil
}

func collectionLength[K, V any](
	tx *vault.Txn,
	collection *vault.Collection[K, V],
) (int, error) {
	length, err := collection.Len(tx)
	if err != nil {
		return 0, fmt.Errorf("read length: %w", err)
	}

	return length, nil
}

var (
	_ PostingIndex    = postingDirectory{}
	_ PostingAdmitter = postingDirectory{}
	_ PostingPurger   = postingDirectory{}
)
