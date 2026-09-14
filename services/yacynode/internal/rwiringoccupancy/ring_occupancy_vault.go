package rwiringoccupancy

import (
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/storedfields"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const ringOccupancyBucket vault.Name = "rwi_ring_occupancy"

func registerRingOccupancy(
	v *vault.Vault,
) (*vault.Collection[yacymodel.DHTRingSector, int], error) {
	occupancyPerSector, err := v.RegisterCollection(
		ringOccupancyBucket,
		dhtRingSectorKeyLayout,
		ringOccupancyValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register ring occupancy of sectors: %w", err)
	}

	return occupancyPerSector, nil
}

var dhtRingSectorKeyLayout = vault.SingleKey(vault.IntegerKeyPart).KeyLayoutFor(
	func(sector yacymodel.DHTRingSector) int64 { return int64(sector) },
	func(sector int64) yacymodel.DHTRingSector { return yacymodel.DHTRingSector(sector) },
)

var errBadRingOccupancy = errors.New("bad ring occupancy of sector")

type ringOccupancyValueCodec struct{}

func (ringOccupancyValueCodec) Encode(amountOfPostings int) ([]byte, error) {
	var stored storedfields.Writer
	stored.Count(amountOfPostings)

	return stored.Record(), nil
}

func (ringOccupancyValueCodec) Decode(raw []byte) (int, error) {
	stored := storedfields.ReaderOf(raw, errBadRingOccupancy)
	amountOfPostings := stored.Count("ring occupancy of sector")

	return amountOfPostings, stored.Err()
}
