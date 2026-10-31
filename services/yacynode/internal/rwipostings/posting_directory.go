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
	postings   *vault.IndexedCollection[postingIdentity, yacymodel.RWIPosting]
	urlWords   *vault.MemberSetIndex[yacymodel.Hash, yacymodel.URLHash, postingIdentity]
	heldProofs heldProofs
	partitions yacymodel.DHTRingPartitions
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
		d.postingsOfWordIn(word, partition),
		func(identity postingIdentity, storedPosting yacymodel.RWIPosting) (bool, error) {
			return visit(postingWithIdentity(identity, storedPosting))
		},
	)
	if err != nil {
		return fmt.Errorf("scan rwi postings of a word in a partition: %w", err)
	}

	return nil
}

func (d postingDirectory) postingsOfWordIn(word yacymodel.Hash, partition uint) vault.KeyRange {
	return postingsOfWordBetween(
		word,
		d.partitions.LowestURLHashIn(partition),
		d.partitions.HighestURLHashIn(partition),
	)
}

func postingWithIdentity(
	identity postingIdentity,
	storedPosting yacymodel.RWIPosting,
) yacymodel.RWIPosting {
	storedPosting.WordHash = identity.word
	storedPosting.URLHash = identity.url

	return storedPosting
}

func (d postingDirectory) Write(tx *vault.Txn, pagePostings yacymodel.PagePostings) error {
	if err := d.postings.PutEach(tx, keyedPostingsOf(pagePostings.Postings())); err != nil {
		return fmt.Errorf("store rwi postings: %w", err)
	}

	return d.heldProofs.mergeWith(tx, pagePostings)
}

func keyedPostingsOf(
	postings []yacymodel.RWIPosting,
) []vault.KeyedValue[postingIdentity, yacymodel.RWIPosting] {
	keyedPostings := make(
		[]vault.KeyedValue[postingIdentity, yacymodel.RWIPosting],
		0,
		len(postings),
	)
	for _, posting := range postings {
		keyedPostings = append(
			keyedPostings,
			vault.KeyedValue[postingIdentity, yacymodel.RWIPosting]{
				Key:   postingIdentity{word: posting.WordHash, url: posting.URLHash},
				Value: posting,
			},
		)
	}

	return keyedPostings
}

func (d postingDirectory) PurgePosting(
	tx *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (wasPurged bool, err error) {
	purgedPosting, wasDeleted, err := d.deletePosting(tx, postingIdentity{word: word, url: url})
	if err != nil || !wasDeleted {
		return false, err
	}
	if err := d.heldProofs.narrowWithout(tx, purgedPosting); err != nil {
		return false, err
	}

	return true, nil
}

func (d postingDirectory) deletePosting(
	tx *vault.Txn,
	identity postingIdentity,
) (deletedPosting yacymodel.RWIPosting, wasDeleted bool, err error) {
	storedPosting, wasDeleted, err := d.postings.DeleteReturning(tx, identity)
	if err != nil {
		return yacymodel.RWIPosting{}, false, fmt.Errorf("delete rwi posting: %w", err)
	}

	return postingWithIdentity(identity, storedPosting), wasDeleted, nil
}

func (d postingDirectory) PurgePostingsOfWordIn(
	tx *vault.Txn,
	word yacymodel.Hash,
	partition uint,
) (amountOfPurgedPostings int, err error) {
	err = d.postings.DeleteWithinVisiting(
		tx,
		d.postingsOfWordIn(word, partition),
		func(identity postingIdentity, storedPosting yacymodel.RWIPosting) error {
			amountOfPurgedPostings++

			return d.heldProofs.narrowWithout(tx, postingWithIdentity(identity, storedPosting))
		},
	)
	if err != nil {
		return 0, fmt.Errorf("delete rwi postings of a word in a partition: %w", err)
	}

	return amountOfPurgedPostings, nil
}

func (d postingDirectory) PurgePostingsOf(
	tx *vault.Txn,
	url yacymodel.URLHash,
) (amountOfPurgedPostings int, err error) {
	words, err := wordsOfURL(tx, d.urlWords, url)
	if err != nil {
		return 0, err
	}
	for _, word := range words {
		wasDeleted, err := d.postings.Delete(tx, postingIdentity{word: word, url: url})
		if err != nil {
			return 0, fmt.Errorf("delete rwi posting: %w", err)
		}
		if wasDeleted {
			amountOfPurgedPostings++
		}
	}
	if err := d.heldProofs.delete(tx, url); err != nil {
		return 0, err
	}

	return amountOfPurgedPostings, nil
}

func wordsOfURL(
	tx *vault.Txn,
	urlWords *vault.MemberSetIndex[yacymodel.Hash, yacymodel.URLHash, postingIdentity],
	url yacymodel.URLHash,
) ([]yacymodel.Hash, error) {
	words, err := urlWords.MembersOf(tx, url)
	if err != nil {
		return nil, fmt.Errorf("read words of a url: %w", err)
	}

	return words, nil
}
