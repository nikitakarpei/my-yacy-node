package distributioncycle

import (
	"slices"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type sectorTurns struct {
	guard           sync.Mutex
	nextSector      yacymodel.DHTRingSector
	sectorsInFlight []yacymodel.DHTRingSector
	drainedSectors  []yacymodel.DHTRingSector
}

func (t *sectorTurns) forgetDrainedSectors() {
	t.guard.Lock()
	defer t.guard.Unlock()

	t.drainedSectors = nil
}

func (t *sectorTurns) claimNext() (yacymodel.DHTRingSector, bool) {
	t.guard.Lock()
	defer t.guard.Unlock()

	for range yacymodel.DHTRingSectorCount {
		sector := t.nextSector
		t.nextSector = (sector + 1) % yacymodel.DHTRingSectorCount
		if slices.Contains(t.sectorsInFlight, sector) || slices.Contains(t.drainedSectors, sector) {
			continue
		}
		t.sectorsInFlight = append(t.sectorsInFlight, sector)

		return sector, true
	}

	return 0, false
}

func (t *sectorTurns) markDrained(sector yacymodel.DHTRingSector) {
	t.guard.Lock()
	defer t.guard.Unlock()

	t.drainedSectors = append(t.drainedSectors, sector)
}

func (t *sectorTurns) release(sector yacymodel.DHTRingSector) {
	t.guard.Lock()
	defer t.guard.Unlock()

	t.sectorsInFlight = slices.DeleteFunc(
		t.sectorsInFlight,
		func(heldSector yacymodel.DHTRingSector) bool { return heldSector == sector },
	)
}
