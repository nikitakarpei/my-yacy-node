package rwiringoccupancy

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ringOccupancy struct {
	occupancyPerSector *vault.Amounts[yacymodel.DHTRingSector]
	partitions         yacymodel.DHTRingPartitions
}

func openRingOccupancy(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (*ringOccupancy, error) {
	occupancyPerSector, err := registerRingOccupancy(v)
	if err != nil {
		return nil, err
	}

	return &ringOccupancy{occupancyPerSector: occupancyPerSector, partitions: partitions}, nil
}

func (o *ringOccupancy) OccupancyOfDHTRingSectors(
	tx *vault.Txn,
) (map[yacymodel.DHTRingSector]int, error) {
	occupancyPerSector := map[yacymodel.DHTRingSector]int{}
	if err := o.occupancyPerSector.Scan(
		tx,
		vault.EveryKey(),
		func(sector yacymodel.DHTRingSector, amountOfPostings int) (bool, error) {
			occupancyPerSector[sector] = amountOfPostings

			return true, nil
		},
	); err != nil {
		return nil, fmt.Errorf("scan ring occupancy of sectors: %w", err)
	}

	return occupancyPerSector, nil
}

func (o *ringOccupancy) PostingStored(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
) error {
	if err := o.occupancyPerSector.Raise(tx, o.dhtRingSectorOf(posting), 1); err != nil {
		return fmt.Errorf("raise ring occupancy of sector: %w", err)
	}

	return nil
}

func (o *ringOccupancy) PostingPurged(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
) error {
	if err := o.occupancyPerSector.Lower(tx, o.dhtRingSectorOf(posting), 1); err != nil {
		return fmt.Errorf("lower ring occupancy of sector: %w", err)
	}

	return nil
}

func (o *ringOccupancy) dhtRingSectorOf(
	posting yacymodel.RWIPosting,
) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(
		yacymodel.DHTRingPositionOfPosting(posting, o.partitions),
	)
}

var _ RingOccupancyProjection = (*ringOccupancy)(nil)
