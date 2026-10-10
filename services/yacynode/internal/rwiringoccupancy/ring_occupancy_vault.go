package rwiringoccupancy

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const ringOccupancyBucket vault.Name = "rwi_ring_occupancy"

func registerRingOccupancy(
	v *vault.Vault,
) (*vault.Amounts[yacymodel.DHTRingSector], error) {
	occupancyPerSector, err := v.RegisterAmounts(ringOccupancyBucket, dhtRingSectorKeyLayout)
	if err != nil {
		return nil, fmt.Errorf("register ring occupancy of sectors: %w", err)
	}

	return occupancyPerSector, nil
}

var dhtRingSectorKeyLayout = vault.SingleKey(vault.IntegerKeyPart).KeyLayoutFor(
	func(sector yacymodel.DHTRingSector) int64 { return int64(sector) },
	func(sector int64) yacymodel.DHTRingSector { return yacymodel.DHTRingSector(sector) },
)
