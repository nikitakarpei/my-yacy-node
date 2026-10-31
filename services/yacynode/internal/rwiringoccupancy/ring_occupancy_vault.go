package rwiringoccupancy

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const ringOccupancyBucket vault.Name = "rwi_ring_occupancy"

const ringOccupancyVersion = "1"

func registerRingOccupancy(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (*vault.CountIndex[yacymodel.RWIPosting, yacymodel.DHTRingSector], error) {
	occupancyPerSector, err := v.RegisterCountIndex(
		ringOccupancyBucket,
		ringOccupancyVersionFor(partitions),
		dhtRingSectorKeyLayout,
		func(posting yacymodel.RWIPosting) yacymodel.DHTRingSector {
			return dhtRingSectorOf(posting, partitions)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("register ring occupancy of sectors: %w", err)
	}

	return occupancyPerSector, nil
}

func ringOccupancyVersionFor(partitions yacymodel.DHTRingPartitions) string {
	return fmt.Sprintf("%s over %d partitions", ringOccupancyVersion, partitions.Amount())
}

func dhtRingSectorOf(
	posting yacymodel.RWIPosting,
	partitions yacymodel.DHTRingPartitions,
) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(yacymodel.DHTRingPositionOfPosting(posting, partitions))
}

var dhtRingSectorKeyLayout = vault.SingleKey(vault.IntegerKeyPart).KeyLayoutFor(
	func(sector yacymodel.DHTRingSector) int64 { return int64(sector) },
	func(sector int64) yacymodel.DHTRingSector { return yacymodel.DHTRingSector(sector) },
)
