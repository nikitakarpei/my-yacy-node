package rwiringoccupancy

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ringOccupancy struct {
	occupancyPerSector *vault.Collection[yacymodel.DHTRingSector, int]
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
	return o.raiseOccupancyByOnePosting(tx, o.dhtRingSectorOf(posting))
}

func (o *ringOccupancy) raiseOccupancyByOnePosting(
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
) error {
	amountOfPostings, err := o.occupancyOfDHTRingSector(tx, sector)
	if err != nil {
		return err
	}
	if _, err := o.occupancyPerSector.Put(tx, sector, amountOfPostings+1); err != nil {
		return fmt.Errorf("record ring occupancy of sector: %w", err)
	}

	return nil
}

func (o *ringOccupancy) PostingPurged(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
) error {
	return o.lowerOccupancyByOnePosting(tx, o.dhtRingSectorOf(posting))
}

func (o *ringOccupancy) lowerOccupancyByOnePosting(
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
) error {
	amountOfPostings, err := o.occupancyOfDHTRingSector(tx, sector)
	if err != nil {
		return err
	}
	if amountOfPostings <= 1 {
		if _, err := o.occupancyPerSector.Delete(tx, sector); err != nil {
			return fmt.Errorf("drop ring occupancy of sector: %w", err)
		}

		return nil
	}
	if _, err := o.occupancyPerSector.Put(tx, sector, amountOfPostings-1); err != nil {
		return fmt.Errorf("record ring occupancy of sector: %w", err)
	}

	return nil
}

func (o *ringOccupancy) occupancyOfDHTRingSector(
	tx *vault.Txn,
	sector yacymodel.DHTRingSector,
) (int, error) {
	amountOfPostings, _, err := o.occupancyPerSector.Get(tx, sector)
	if err != nil {
		return 0, fmt.Errorf("read ring occupancy of sector: %w", err)
	}

	return amountOfPostings, nil
}

func (o *ringOccupancy) dhtRingSectorOf(
	posting yacymodel.RWIPosting,
) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(
		yacymodel.DHTRingPositionOfPosting(posting, o.partitions),
	)
}

var _ RingOccupancyProjection = (*ringOccupancy)(nil)
