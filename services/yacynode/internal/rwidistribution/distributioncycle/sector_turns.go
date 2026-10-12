package distributioncycle

import (
	"slices"
	"sync"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/wordpartitionoffer"
)

type sectorTurns struct {
	guard           sync.Mutex
	nextSector      yacymodel.DHTRingSector
	sectorsInFlight []yacymodel.DHTRingSector
	drainedSectors  []yacymodel.DHTRingSector
	cursors         map[yacymodel.DHTRingSector]wordpartitionoffer.Cursor
}

func newSectorTurns() *sectorTurns {
	return &sectorTurns{cursors: make(map[yacymodel.DHTRingSector]wordpartitionoffer.Cursor)}
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

func (t *sectorTurns) cursorOf(sector yacymodel.DHTRingSector) wordpartitionoffer.Cursor {
	t.guard.Lock()
	defer t.guard.Unlock()

	return t.cursors[sector]
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

func (t *sectorTurns) advance(sector yacymodel.DHTRingSector, next wordpartitionoffer.Cursor) {
	t.guard.Lock()
	defer t.guard.Unlock()

	t.cursors[sector] = next
}
