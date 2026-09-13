package rwipostingsectoramount

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingSectorAmounts struct {
	amountPerSector *vault.Collection[yacymodel.DHTRingSector, int]
	partitions      yacymodel.DHTRingPartitions
}

func openPostingSectorAmounts(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (*postingSectorAmounts, error) {
	amountPerSector, err := registerPostingSectorAmounts(v)
	if err != nil {
		return nil, err
	}

	return &postingSectorAmounts{amountPerSector: amountPerSector, partitions: partitions}, nil
}

func (a *postingSectorAmounts) AmountOfPostingsPerDHTRingSector(
	tx *vault.Txn,
) (map[yacymodel.DHTRingSector]int, error) {
	amountPerSector := map[yacymodel.DHTRingSector]int{}
	if err := a.amountPerSector.Scan(
		tx,
		vault.EveryKey(),
		func(sector yacymodel.DHTRingSector, amountOfPostings int) (bool, error) {
			amountPerSector[sector] = amountOfPostings

			return true, nil
		},
	); err != nil {
		return nil, fmt.Errorf("scan posting amounts per sector: %w", err)
	}

	return amountPerSector, nil
}

func (a *postingSectorAmounts) PostingStored(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
) error {
	return a.countOnePostingMore(tx, a.dhtRingSectorOf(posting))
}

func (a *postingSectorAmounts) countOnePostingMore(
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
) error {
	amountOfPostings, err := a.amountOfPostingsOf(tx, sector)
	if err != nil {
		return err
	}
	if _, err := a.amountPerSector.Put(tx, sector, amountOfPostings+1); err != nil {
		return fmt.Errorf("record posting amount of sector: %w", err)
	}

	return nil
}

func (a *postingSectorAmounts) PostingPurged(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
) error {
	return a.countOnePostingLess(tx, a.dhtRingSectorOf(posting))
}

func (a *postingSectorAmounts) countOnePostingLess(
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
) error {
	amountOfPostings, err := a.amountOfPostingsOf(tx, sector)
	if err != nil {
		return err
	}
	if amountOfPostings <= 1 {
		if _, err := a.amountPerSector.Delete(tx, sector); err != nil {
			return fmt.Errorf("drop posting amount of sector: %w", err)
		}

		return nil
	}
	if _, err := a.amountPerSector.Put(tx, sector, amountOfPostings-1); err != nil {
		return fmt.Errorf("record posting amount of sector: %w", err)
	}

	return nil
}

func (a *postingSectorAmounts) amountOfPostingsOf(
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
) (int, error) {
	amountOfPostings, _, err := a.amountPerSector.Get(tx, sector)
	if err != nil {
		return 0, fmt.Errorf("read posting amount of sector: %w", err)
	}

	return amountOfPostings, nil
}

func (a *postingSectorAmounts) dhtRingSectorOf(
	posting yacymodel.RWIPosting,
) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(
		yacymodel.DHTRingPositionOfPosting(posting, a.partitions),
	)
}

var _ PostingSectorAmountProjection = (*postingSectorAmounts)(nil)
