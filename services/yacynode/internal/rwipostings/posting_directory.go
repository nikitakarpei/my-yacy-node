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
	postings      *vault.Collection[postingIdentity, yacymodel.RWIPosting]
	urlReferences *vault.SecondPartIndex[yacymodel.Hash, yacymodel.URLHash, postingIdentity]
	partitions    yacymodel.DHTRingPartitions
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

func (d postingDirectory) WordsReferencing(
	tx *vault.Txn,
	url yacymodel.URLHash,
) ([]yacymodel.Hash, error) {
	var words []yacymodel.Hash
	if err := d.urlReferences.Scan(tx, url, func(identity postingIdentity) (bool, error) {
		words = append(words, identity.word)

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("scan words referencing a url: %w", err)
	}

	return words, nil
}

func (d postingDirectory) Admit(tx *vault.Txn, postings ...yacymodel.RWIPosting) error {
	if err := d.postings.PutEach(tx, keyedPostingsOf(postings)); err != nil {
		return fmt.Errorf("store rwi postings: %w", err)
	}

	return nil
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
	identity := postingIdentity{word: word, url: url}
	wasDeleted, err := d.postings.Delete(tx, identity)
	if err != nil {
		return false, fmt.Errorf("delete rwi posting: %w", err)
	}

	return wasDeleted, nil
}

var (
	_ PostingIndex    = postingDirectory{}
	_ URLReferences   = postingDirectory{}
	_ PostingAdmitter = postingDirectory{}
	_ PostingPurger   = postingDirectory{}
)
