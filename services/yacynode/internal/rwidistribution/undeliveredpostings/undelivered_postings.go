// Package undeliveredpostings owns the record of the postings that were added
// or replaced and are not yet delivered to all their responsible peers. The
// record follows the ring: a read takes the next word partitions after the last
// posting read, up to a budget of postings, never splits a word partition, and
// tells when it reached the end of the record.
package undeliveredpostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type UndeliveredPostings struct {
	postings   *vault.ChangeIndex[yacymodel.RWIPosting, undeliveredEntry]
	partitions yacymodel.DHTRingPartitions
}

func Open(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (*UndeliveredPostings, vault.Projection[yacymodel.RWIPosting], error) {
	postings, err := registerUndeliveredPostings(v, partitions)
	if err != nil {
		return nil, nil, err
	}

	return &UndeliveredPostings{postings: postings, partitions: partitions}, postings, nil
}

func (u *UndeliveredPostings) WordPartitionsAfter(
	tx *vault.Txn,
	lastReadPosting UndeliveredPosting,
	postingsBudget int,
) (wordPartitions []UndeliveredWordPartition, reachedEnd bool, err error) {
	reading := wordPartitionReadingWithin(postingsBudget)
	if err := u.postings.Scan(
		tx,
		entriesAfter(lastReadPosting),
		func(change vault.Change[undeliveredEntry]) (bool, error) {
			return reading.visit(undeliveredPostingFrom(change, u.partitions)), nil
		},
	); err != nil {
		return nil, false, fmt.Errorf("read undelivered word partitions: %w", err)
	}

	return reading.wordPartitions(), !reading.stoppedOnBudget, nil
}

func (u *UndeliveredPostings) MarkDelivered(
	tx *vault.Txn,
	posting UndeliveredPosting,
) (wasMarked bool, err error) {
	wasMarked, err = u.postings.Acknowledge(tx, posting.change)
	if err != nil {
		return false, fmt.Errorf("mark posting delivered: %w", err)
	}

	return wasMarked, nil
}

func (u *UndeliveredPostings) PostingsOf(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
	visit func(UndeliveredPosting) (bool, error),
) error {
	if err := u.postings.Scan(
		tx,
		entriesOf(wordPartition, u.partitions),
		func(change vault.Change[undeliveredEntry]) (bool, error) {
			return visit(undeliveredPostingFrom(change, u.partitions))
		},
	); err != nil {
		return fmt.Errorf("read undelivered postings of a word partition: %w", err)
	}

	return nil
}

func (u *UndeliveredPostings) AnyIn(
	tx *vault.Txn,
	wordPartition yacymodel.WordPartition,
) (bool, error) {
	var found bool
	if err := u.postings.Scan(
		tx,
		entriesOf(wordPartition, u.partitions),
		func(vault.Change[undeliveredEntry]) (bool, error) {
			found = true

			return false, nil
		},
	); err != nil {
		return false, fmt.Errorf("find undelivered postings of a word partition: %w", err)
	}

	return found, nil
}

func (u *UndeliveredPostings) AmountOfPostings(tx *vault.Txn) (int, error) {
	amountOfPostings, err := u.postings.Len(tx)
	if err != nil {
		return 0, fmt.Errorf("count undelivered postings: %w", err)
	}

	return amountOfPostings, nil
}
