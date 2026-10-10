package distributioncycle

import (
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type sectorsInFlight struct {
	claimTurn chan struct{}
	sectors   []yacymodel.DHTRingSector
}

func newSectorsInFlight() *sectorsInFlight {
	return &sectorsInFlight{claimTurn: make(chan struct{}, 1)}
}

func (f *sectorsInFlight) claimFree(
	pick func(sectorsInFlight []yacymodel.DHTRingSector) (yacymodel.DHTRingSector, bool, error),
) (yacymodel.DHTRingSector, bool, error) {
	f.takeClaimTurn()
	defer f.endClaimTurn()

	sector, found, err := pick(f.sectors)
	if err != nil || !found {
		return 0, false, err
	}
	f.sectors = append(f.sectors, sector)

	return sector, true, nil
}

func (f *sectorsInFlight) takeClaimTurn() { f.claimTurn <- struct{}{} }

func (f *sectorsInFlight) endClaimTurn() { <-f.claimTurn }

func (f *sectorsInFlight) release(sector yacymodel.DHTRingSector) {
	f.takeClaimTurn()
	defer f.endClaimTurn()

	f.sectors = slices.DeleteFunc(f.sectors, func(heldSector yacymodel.DHTRingSector) bool {
		return heldSector == sector
	})
}
