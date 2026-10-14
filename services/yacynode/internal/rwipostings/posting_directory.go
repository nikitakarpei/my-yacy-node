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
	postings   *vault.Collection[postingIdentity, yacymodel.RWIPosting]
	partitions yacymodel.DHTRingPartitions
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

func (d postingDirectory) PostingsOfWordIn(
	tx *vault.Txn,
	word yacymodel.Hash,
	partition uint,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	err := d.postings.Scan(
		tx,
		postingsOfWordBetween(
			word,
			d.partitions.LowestURLHashIn(partition),
			d.partitions.HighestURLHashIn(partition),
		),
		func(identity postingIdentity, storedPosting yacymodel.RWIPosting) (bool, error) {
			return visit(postingWithIdentity(identity, storedPosting))
		},
	)
	if err != nil {
		return fmt.Errorf("scan rwi postings of a word in a partition: %w", err)
	}

	return nil
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
	if _, err := d.postings.Put(tx, identity, posting); err != nil {
		return fmt.Errorf("store rwi posting: %w", err)
	}

	return nil
}

func (d postingDirectory) PurgePosting(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (wasPurged bool, err error) {
	identity := postingIdentity{word: word, url: url}
	wasDeleted, err := d.postings.Delete(tx, identity)
	if err != nil {
		return false, fmt.Errorf("delete rwi posting: %w", err)
	}

	return wasDeleted, nil
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
