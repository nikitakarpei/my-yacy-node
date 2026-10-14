package wordpartitionschedule

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const admissionsBucket vault.Name = "rwidistribution_unscheduled_admissions"

type admissions struct {
	amounts    *vault.Amounts[yacymodel.WordPartition]
	partitions yacymodel.DHTRingPartitions
}

type admittedWordPartition struct {
	wordPartition      yacymodel.WordPartition
	amountOfAdmissions int
}

func openAdmissions(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (*admissions, error) {
	amounts, err := v.RegisterAmounts(admissionsBucket, admissionKeyLayoutFor(partitions))
	if err != nil {
		return nil, fmt.Errorf("register unscheduled admissions: %w", err)
	}

	return &admissions{amounts: amounts, partitions: partitions}, nil
}

var admissionKeyParts = vault.TripleKey(
	vault.IntegerKeyPart,
	hashkeypart.Hash,
	vault.IntegerKeyPart,
)

func admissionKeyLayoutFor(
	partitions yacymodel.DHTRingPartitions,
) vault.KeyLayout[yacymodel.WordPartition] {
	return admissionKeyParts.KeyLayoutFor(
		func(wordPartition yacymodel.WordPartition) (int64, yacymodel.Hash, int64) {
			slice := scheduleSliceOf(
				yacymodel.DHTRingPositionOfWordPartition(wordPartition, partitions),
			)

			return sliceKeyOf(slice), wordPartition.Word, partitionKeyOf(wordPartition.Partition)
		},
		func(_ int64, word yacymodel.Hash, partition int64) yacymodel.WordPartition {
			return yacymodel.WordPartition{Word: word, Partition: partitionFromKey(partition)}
		},
	)
}

func (a *admissions) count(tx *vault.Txn, wordPartition yacymodel.WordPartition) error {
	if err := a.amounts.Raise(tx, wordPartition, 1); err != nil {
		return fmt.Errorf("count admission: %w", err)
	}

	return nil
}

func (a *admissions) within(
	tx *vault.Txn,
	span yacymodel.DHTRingSpan,
) ([]admittedWordPartition, error) {
	var admitted []admittedWordPartition
	for slice := scheduleSliceOf(span.First); slice <= scheduleSliceOf(span.Last); slice++ {
		if err := a.amounts.Scan(
			tx,
			admissionKeyParts.KeysWithFirst(sliceKeyOf(slice)),
			func(wordPartition yacymodel.WordPartition, amountOfAdmissions int) (bool, error) {
				if span.Contains(
					yacymodel.DHTRingPositionOfWordPartition(wordPartition, a.partitions),
				) {
					admitted = append(admitted, admittedWordPartition{
						wordPartition:      wordPartition,
						amountOfAdmissions: amountOfAdmissions,
					})
				}

				return true, nil
			},
		); err != nil {
			return nil, fmt.Errorf("read unscheduled admissions: %w", err)
		}
	}

	return admitted, nil
}

func (a *admissions) markScheduled(tx *vault.Txn, admitted admittedWordPartition) error {
	if err := a.amounts.Lower(
		tx,
		admitted.wordPartition,
		uint(admitted.amountOfAdmissions),
	); err != nil {
		return fmt.Errorf("mark admissions scheduled: %w", err)
	}

	return nil
}
