package rwiringoccupancy

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ringOccupancy struct {
	occupancyPerSector *vault.CountIndex[yacymodel.RWIPosting, yacymodel.DHTRingSector]
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

var _ RingOccupancyQuery = (*ringOccupancy)(nil)
