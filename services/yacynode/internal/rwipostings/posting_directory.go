package rwipostings

import (
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
	identity := postingIdentity{word: word, url: url}
	storedPosting, found, err := d.postings.Get(tx, identity)
	if err != nil {
		return yacymodel.RWIPosting{}, false, fmt.Errorf("read rwi posting: %w", err)
	}
	if !found {
		return yacymodel.RWIPosting{}, false, nil
	}

	return postingWithIdentity(identity, storedPosting), true, nil
}

func postingWithIdentity(
	identity postingIdentity,
	storedPosting yacymodel.RWIPosting,
) yacymodel.RWIPosting {
	storedPosting.WordHash = identity.word
	storedPosting.URLHash = identity.url

	return storedPosting
}

func (d postingDirectory) Admit(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	identity := postingIdentity{word: posting.WordHash, url: posting.URLHash}
	previousPosting, wasReplaced, err := d.postings.PutReturning(tx, identity, posting)
	if err != nil {
		return fmt.Errorf("store rwi posting: %w", err)
	}
	if !wasReplaced {
		return d.observers.stored(tx, posting)
	}

	return d.observers.updated(tx, postingWithIdentity(identity, previousPosting), posting)
}

func (d postingDirectory) PurgePosting(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (wasPurged bool, err error) {
	identity := postingIdentity{word: word, url: url}
	purgedPosting, wasDeleted, err := d.postings.DeleteReturning(tx, identity)
	if err != nil {
		return false, fmt.Errorf("delete rwi posting: %w", err)
	}
	if !wasDeleted {
		return false, nil
	}
	if err := d.observers.purged(tx, postingWithIdentity(identity, purgedPosting)); err != nil {
		return false, err
	}

	return true, nil
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
